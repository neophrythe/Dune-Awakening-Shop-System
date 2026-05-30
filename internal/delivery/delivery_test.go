package delivery

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBuildEnvelope(t *testing.T) {
	b64, err := buildEnvelope(map[string]any{"ServerCommand": "AddItemToInventory", "Quantity": 2})
	if err != nil {
		t.Fatalf("buildEnvelope: %v", err)
	}
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	var env struct {
		Version        int
		AuthToken      string
		MessageContent string
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	if env.Version != 2 || env.AuthToken != serverCmdAuthToken {
		t.Fatalf("bad envelope header: %+v", env)
	}
	var inner map[string]any
	if err := json.Unmarshal([]byte(env.MessageContent), &inner); err != nil {
		t.Fatalf("unmarshal inner: %v", err)
	}
	if inner["ServerCommand"] != "AddItemToInventory" {
		t.Fatalf("inner command = %v", inner["ServerCommand"])
	}
}

type fakeExecer struct {
	name string
	args []string
}

func (f *fakeExecer) Run(_ context.Context, name string, args ...string) (string, error) {
	f.name, f.args = name, args
	return "", nil
}

func TestRMQEngineDeliver(t *testing.T) {
	fe := &fakeExecer{}
	e := &RMQEngine{Container: "AMP_X", Exec: fe, NowMilli: func() int64 { return 123 }}
	if err := e.Deliver(context.Background(),
		Request{PlayFabID: "A93638E049FBE2D9", AssetItemID: "Ammo", Count: 3}); err != nil {
		t.Fatalf("deliver: %v", err)
	}
	if fe.name != "docker" || len(fe.args) != 5 {
		t.Fatalf("unexpected exec: %s %v", fe.name, fe.args)
	}
	if fe.args[0] != "exec" || fe.args[1] != "AMP_X" || fe.args[2] != "sh" || fe.args[3] != "-c" {
		t.Fatalf("unexpected docker args: %v", fe.args[:4])
	}
	inner := fe.args[4]
	if !strings.Contains(inner, "rabbitmqctl --node rabbit-game@localhost eval") {
		t.Fatalf("missing rabbitmqctl eval: %s", inner)
	}
	if !strings.Contains(inner, "dune-shop-cmd-123") {
		t.Fatalf("missing message id: %s", inner)
	}
	wantEnv, _ := buildEnvelope(map[string]any{
		"ServerCommand": "AddItemToInventory", "PlayerId": "A93638E049FBE2D9",
		"ItemName": "Ammo", "Quantity": 3, "Durability": 1.0,
	})
	if !strings.Contains(inner, wantEnv) {
		t.Fatalf("missing expected envelope base64 in command")
	}
}

func TestRMQEngineValidation(t *testing.T) {
	e := &RMQEngine{Exec: &fakeExecer{}}
	if err := e.Deliver(context.Background(), Request{PlayFabID: "x", AssetItemID: "y"}); err == nil {
		t.Fatal("expected error for missing container")
	}
	e.Container = "AMP_X"
	if err := e.Deliver(context.Background(), Request{AssetItemID: "y"}); err == nil {
		t.Fatal("expected error for missing player id")
	}
}

func TestFLSEngineDeliver(t *testing.T) {
	var gotPath, gotKey string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotKey = r.Header.Get("X-SecretKey")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":200}`))
	}))
	defer srv.Close()

	e := &FLSEngine{Token: "tok", TitleID: "ABC", baseURL: srv.URL}
	if err := e.Deliver(context.Background(),
		Request{PlayFabID: "pf1", PlayFabItemID: "itm", Count: 2}); err != nil {
		t.Fatalf("deliver: %v", err)
	}
	if gotPath != "/Server/GrantItemsToUser" {
		t.Fatalf("path = %s", gotPath)
	}
	if gotKey != "tok" {
		t.Fatalf("secret key = %s", gotKey)
	}
	if gotBody["PlayFabId"] != "pf1" || gotBody["CatalogVersion"] != "Live" {
		t.Fatalf("body = %+v", gotBody)
	}
	if ids, ok := gotBody["ItemIds"].([]any); !ok || len(ids) != 2 {
		t.Fatalf("ItemIds = %v", gotBody["ItemIds"])
	}
}

func TestFLSEngineErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"nope"}`))
	}))
	defer srv.Close()
	e := &FLSEngine{Token: "tok", TitleID: "ABC", baseURL: srv.URL}
	if err := e.Deliver(context.Background(),
		Request{PlayFabID: "pf1", PlayFabItemID: "itm"}); err == nil {
		t.Fatal("expected error on non-200")
	}
}

type fixedEngine struct {
	name string
	err  error
}

func (f fixedEngine) Name() string                           { return f.name }
func (f fixedEngine) Deliver(context.Context, Request) error { return f.err }

func TestMulti(t *testing.T) {
	ok := fixedEngine{name: "ok"}
	bad := fixedEngine{name: "bad", err: errors.New("boom")}

	if err := (&Multi{Engines: []Engine{bad, ok}}).Deliver(context.Background(), Request{}); err != nil {
		t.Fatalf("expected first-success, got %v", err)
	}
	if err := (&Multi{Engines: []Engine{bad, bad}}).Deliver(context.Background(), Request{}); err == nil {
		t.Fatal("expected failure when all engines fail")
	}
}

func TestNewModes(t *testing.T) {
	for _, m := range []string{"", "fls", "rmq", "both"} {
		if _, err := New(Options{Mode: m, Container: "c", FLSToken: "t", PlayFabTitleID: "x"}); err != nil {
			t.Fatalf("mode %q: %v", m, err)
		}
	}
	if _, err := New(Options{Mode: "bogus"}); err == nil {
		t.Fatal("expected error for unknown mode")
	}
}
