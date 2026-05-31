// Package backend is the external destination that filed findings are
// mirrored to. The in-memory backend is the zero-dependency default; the
// webhook backend POSTs each filed ticket to a configured URL.
//
// The plugin always keeps its own authoritative ticket list in memory
// (see the tickets package) so ListTickets/Annotate work regardless of
// backend; the Sink only decides where a *newly filed* ticket is pushed
// externally.
package backend

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	pb "github.com/zeroroot-ai/gibson-redteam-example/findings-sink/api/gen/gibson/plugins/findingssink/v1"
)

// Sink is an external destination for filed tickets.
type Sink interface {
	// Save mirrors a filed ticket to the external destination.
	Save(ctx context.Context, t *pb.Ticket) error
	// Name identifies the backend for logging.
	Name() string
}

// InMemory is the default backend: it keeps nothing externally. The
// plugin's own ticket store remains authoritative.
type InMemory struct{}

func (InMemory) Save(context.Context, *pb.Ticket) error { return nil }
func (InMemory) Name() string                            { return "in-memory" }

// Webhook POSTs each filed ticket as JSON to URL, optionally
// authenticated with a bearer Token.
type Webhook struct {
	URL    string
	Token  string
	Client *http.Client
}

func (w Webhook) Name() string { return "webhook" }

func (w Webhook) Save(ctx context.Context, t *pb.Ticket) error {
	body, err := protojson.Marshal(t)
	if err != nil {
		return fmt.Errorf("marshal ticket: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if w.Token != "" {
		req.Header.Set("Authorization", "Bearer "+w.Token)
	}
	client := w.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return nil
}

// Select returns a Webhook backend when webhookURL is non-empty,
// otherwise the in-memory default. token is attached as a bearer
// credential to webhook requests.
func Select(webhookURL, token string) Sink {
	if webhookURL == "" {
		return InMemory{}
	}
	return Webhook{URL: webhookURL, Token: token, Client: &http.Client{Timeout: 10 * time.Second}}
}
