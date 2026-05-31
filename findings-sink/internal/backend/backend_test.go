package backend_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zeroroot-ai/gibson-redteam-example/findings-sink/internal/backend"
	pb "github.com/zeroroot-ai/gibson-redteam-example/findings-sink/api/gen/gibson/plugins/findingssink/v1"
)

func TestSelect(t *testing.T) {
	if got := backend.Select("", ""); got.Name() != "in-memory" {
		t.Errorf("empty url → %q, want in-memory", got.Name())
	}
	if got := backend.Select("https://hook.example", "tok"); got.Name() != "webhook" {
		t.Errorf("url set → %q, want webhook", got.Name())
	}
}

func TestInMemorySaveIsNoop(t *testing.T) {
	if err := (backend.InMemory{}).Save(context.Background(), &pb.Ticket{TicketId: "x"}); err != nil {
		t.Errorf("in-memory Save: %v", err)
	}
}

func TestWebhookSave(t *testing.T) {
	var gotAuth, gotCT string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotCT = r.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	wh := backend.Webhook{URL: srv.URL, Token: "sekret", Client: srv.Client()}
	err := wh.Save(context.Background(), &pb.Ticket{
		TicketId: "FS-1",
		Finding:  &pb.Finding{Title: "leak", Severity: "high"},
	})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if gotAuth != "Bearer sekret" {
		t.Errorf("auth header = %q", gotAuth)
	}
	if gotCT != "application/json" {
		t.Errorf("content-type = %q", gotCT)
	}
	if len(gotBody) == 0 {
		t.Errorf("expected a JSON body")
	}
}

func TestWebhookSave_non2xxIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	wh := backend.Webhook{URL: srv.URL, Client: srv.Client()}
	if err := wh.Save(context.Background(), &pb.Ticket{TicketId: "FS-1"}); err == nil {
		t.Errorf("expected error on 500 response")
	}
}
