package delivery

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// serverCmdAuthToken is the static AuthToken the game server validates on
// incoming server-command envelopes (from dune-admin / send-dune-broadcast).
const serverCmdAuthToken = "Nu6VmPWUMvdPMeB7qErr"

const (
	defaultMQRoot = "/AMP/duneawakening/extracted/mq"
	defaultMQHome = "/AMP/duneawakening/runtime/mq-game-home"
	defaultNode   = "rabbit-game@localhost"
)

// Execer runs a command and returns its combined output. Abstracted for tests.
type Execer interface {
	Run(ctx context.Context, name string, args ...string) (string, error)
}

type osExecer struct{}

func (osExecer) Run(ctx context.Context, name string, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	return string(out), err
}

// RMQEngine delivers items live by publishing a SpawnItem server command to the
// running game server's RabbitMQ broker inside the AMP container. Immediate, but
// the target player must be online.
type RMQEngine struct {
	Container string // docker container, e.g. AMP_BuGIsland01
	MQRoot    string // extracted mq dir
	Home      string // broker HOME containing .erlang.cookie
	Node      string // broker node name
	Exec      Execer // defaults to os/exec
	NowMilli  func() int64
}

// Name implements Engine.
func (e *RMQEngine) Name() string { return "rmq" }

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// Deliver publishes a SpawnItem server command for the player.
func (e *RMQEngine) Deliver(ctx context.Context, r Request) error {
	if e.Container == "" {
		return fmt.Errorf("rmq deliver: no container configured")
	}
	if r.PlayerName == "" || r.AssetItemID == "" {
		return fmt.Errorf("rmq deliver: missing player name or item id")
	}
	count := r.Count
	if count < 1 {
		count = 1
	}
	envB64, err := buildEnvelope(map[string]any{
		"Command":    "SpawnItem",
		"PlayerName": r.PlayerName,
		"ItemId":     r.AssetItemID,
		"Count":      count,
	})
	if err != nil {
		return err
	}

	now := e.NowMilli
	if now == nil {
		now = func() int64 { return time.Now().UnixMilli() }
	}
	erlang := erlangPublish(envB64, fmt.Sprintf("dune-shop-cmd-%d", now()))
	inner := e.rabbitmqctlEval(erlang)

	execer := e.Exec
	if execer == nil {
		execer = osExecer{}
	}
	out, err := execer.Run(ctx, "docker", "exec", e.Container, "sh", "-c", inner)
	if err != nil {
		return fmt.Errorf("rmq deliver: %w (output: %s)", err, strings.TrimSpace(out))
	}
	return nil
}

// rabbitmqctl Eval invocation, run inside the container against the musl-linked
// erlang/rabbitmq bundled with the Funcom server image.
func (e *RMQEngine) rabbitmqctlEval(erlang string) string {
	mq := orDefault(e.MQRoot, defaultMQRoot)
	home := orDefault(e.Home, defaultMQHome)
	node := orDefault(e.Node, defaultNode)
	return fmt.Sprintf(
		`env -i HOME=%s LC_ALL=C `+
			`LD_LIBRARY_PATH=%[2]s/lib:%[2]s/usr/lib:%[2]s/opt/openssl/lib `+
			`RABBITMQ_HOME=%[2]s/opt/rabbitmq `+
			`%[2]s/lib/ld-musl-x86_64.so.1 `+
			`%[2]s/opt/erlang/lib/erlang/bin/escript `+
			`%[2]s/opt/rabbitmq/escript/rabbitmqctl --node %s eval %s`,
		home, mq, node, shellQuote(erlang))
}

// buildEnvelope marshals the base64 server-command envelope (Version 2 +
// AuthToken + JSON MessageContent), mirroring dune-admin's publishServerCommand.
func buildEnvelope(fields map[string]any) (string, error) {
	inner, err := json.Marshal(fields)
	if err != nil {
		return "", fmt.Errorf("marshal command: %w", err)
	}
	outer, err := json.Marshal(map[string]any{
		"Version":        2,
		"AuthToken":      serverCmdAuthToken,
		"MessageContent": string(inner),
	})
	if err != nil {
		return "", fmt.Errorf("marshal envelope: %w", err)
	}
	return base64.StdEncoding.EncodeToString(outer), nil
}

// erlangPublish builds the rabbitmqctl eval expression that publishes the base64
// envelope to the heartbeats exchange with user_id=fls (which AMQP clients can't
// set), routing key "notifications".
func erlangPublish(envelopeB64, msgID string) string {
	return fmt.Sprintf(
		`Outer = base64:decode(<<"%s">>),`+
			`XName = rabbit_misc:r(<<"/">>, exchange, <<"heartbeats">>),`+
			`X = rabbit_exchange:lookup_or_die(XName),`+
			`MsgId = <<"%s">>,`+
			`P = {list_to_atom("P_basic"), <<"Content">>, undefined, [], undefined,`+
			` undefined, undefined, undefined, undefined, MsgId, undefined,`+
			` undefined, <<"fls">>, <<"fls_backend">>, undefined},`+
			`Content = rabbit_basic:build_content(P, Outer),`+
			`{ok, Msg} = rabbit_basic:message(XName, <<"notifications">>, Content),`+
			`rabbit_queue_type:publish_at_most_once(X, Msg).`,
		envelopeB64, msgID)
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
