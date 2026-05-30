package delivery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// FLSEngine grants items account-level via Funcom Live Services → PlayFab
// Server/GrantItemsToUser. The ServiceAuthToken (self-host JWT) is sent as the
// PlayFab X-SecretKey. Items appear in the player's inventory on next login and
// work even if the player is offline.
type FLSEngine struct {
	Token   string       // ServiceAuthToken (self-host JWT)
	TitleID string       // PlayFab title id
	Client  *http.Client // defaults to http.DefaultClient

	baseURL string // test override; empty = https://<TitleID>.playfabapi.com
}

// Name implements Engine.
func (e *FLSEngine) Name() string { return "fls" }

func (e *FLSEngine) endpoint() string {
	if e.baseURL != "" {
		return e.baseURL
	}
	return fmt.Sprintf("https://%s.playfabapi.com", e.TitleID)
}

// Deliver grants Count copies of the PlayFab catalog item to the account.
func (e *FLSEngine) Deliver(ctx context.Context, r Request) error {
	if e.Token == "" || e.TitleID == "" {
		return fmt.Errorf("fls deliver: missing token or title id")
	}
	if r.PlayFabID == "" || r.PlayFabItemID == "" {
		return fmt.Errorf("fls deliver: missing playfab id or item id")
	}
	count := r.Count
	if count < 1 {
		count = 1
	}
	ids := make([]string, count)
	for i := range ids {
		ids[i] = r.PlayFabItemID
	}
	body, err := json.Marshal(map[string]any{
		"PlayFabId":      r.PlayFabID,
		"ItemIds":        ids,
		"CatalogVersion": "Live",
	})
	if err != nil {
		return fmt.Errorf("fls deliver: marshal: %w", err)
	}

	url := e.endpoint() + "/Server/GrantItemsToUser"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("fls deliver: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-SecretKey", e.Token)

	client := e.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("fls deliver: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("fls deliver: status %d: %s", resp.StatusCode, string(b))
	}
	return nil
}
