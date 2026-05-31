// Command findings-sink is the stateful plugin of the red-team
// reference trio. It tracks findings the agent files and exposes three
// methods: FileFinding, ListTickets, and Annotate.
//
// The plugin keeps an authoritative in-process ticket registry. A
// backend Sink decides where a newly filed ticket is *also* pushed: the
// zero-dependency in-memory default, or a webhook when
// FINDINGS_SINK_WEBHOOK_URL is configured. The webhook is authenticated
// with the cred:tracker-token secret declared in plugin.yaml.
//
// See AGENTS.md for the full Gibson plugin contract.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/zeroroot-ai/sdk/plugin"
	"github.com/zeroroot-ai/sdk/plugin/lifecycle"
	"google.golang.org/protobuf/proto"

	pb "github.com/zeroroot-ai/gibson-redteam-example/findings-sink/api/gen/gibson/plugins/findingssink/v1"
	"github.com/zeroroot-ai/gibson-redteam-example/findings-sink/internal/backend"
	"github.com/zeroroot-ai/gibson-redteam-example/findings-sink/internal/tickets"
)

const (
	envWebhookURL   = "FINDINGS_SINK_WEBHOOK_URL"
	envTrackerToken = "FINDINGS_SINK_TRACKER_TOKEN"
)

// FindingsSink holds the plugin's ticket registry.
type FindingsSink struct {
	store *tickets.Store
}

// newFindingsSink starts with the safe in-memory default so handlers
// never see a nil store even before OnStart runs.
func newFindingsSink() *FindingsSink {
	return &FindingsSink{store: tickets.NewStore(backend.InMemory{})}
}

// onStart selects the backend from configuration. The webhook is used
// only when FINDINGS_SINK_WEBHOOK_URL is set; otherwise the in-memory
// default keeps the plugin dependency-free.
func (fs *FindingsSink) onStart(ctx context.Context) error {
	sink := backend.Select(os.Getenv(envWebhookURL), os.Getenv(envTrackerToken))
	fs.store = tickets.NewStore(sink)
	slog.InfoContext(ctx, "findings-sink ready", "backend", sink.Name())
	return nil
}

func (fs *FindingsSink) fileFinding(ctx context.Context, req proto.Message) (proto.Message, error) {
	r, ok := req.(*pb.FileFindingRequest)
	if !ok {
		return nil, fmt.Errorf("FileFinding: unexpected request type %T", req)
	}
	id, err := fs.store.File(ctx, r.GetFinding())
	if err != nil {
		// The ticket is filed locally; only the external mirror failed.
		slog.WarnContext(ctx, "ticket filed but external mirror failed", "ticket", id, "err", err)
	}
	return &pb.FileFindingResponse{TicketId: id}, nil
}

func (fs *FindingsSink) listTickets(_ context.Context, req proto.Message) (proto.Message, error) {
	r, ok := req.(*pb.ListTicketsRequest)
	if !ok {
		return nil, fmt.Errorf("ListTickets: unexpected request type %T", req)
	}
	return &pb.ListTicketsResponse{Tickets: fs.store.List(r.GetSeverity(), r.GetCategory())}, nil
}

func (fs *FindingsSink) annotate(_ context.Context, req proto.Message) (proto.Message, error) {
	r, ok := req.(*pb.AnnotateRequest)
	if !ok {
		return nil, fmt.Errorf("Annotate: unexpected request type %T", req)
	}
	return &pb.AnnotateResponse{Ok: fs.store.Annotate(r.GetTicketId(), r.GetNote())}, nil
}

func main() {
	fs := newFindingsSink()
	err := plugin.Serve(
		context.Background(),
		plugin.WithManifest("./plugin.yaml"),
		plugin.WithLifecycle(lifecycle.LifecycleHooks{OnStart: fs.onStart}),
		plugin.WithMethod("FileFinding", fs.fileFinding),
		plugin.WithMethod("ListTickets", fs.listTickets),
		plugin.WithMethod("Annotate", fs.annotate),
	)
	if err != nil {
		slog.Error("plugin exited with error", "err", err)
		os.Exit(1)
	}
}
