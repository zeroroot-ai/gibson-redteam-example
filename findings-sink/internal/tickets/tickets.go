// Package tickets is the plugin's authoritative, in-process ticket
// registry. Filing a finding mints a ticket id, stores it, and mirrors
// it to the configured backend Sink. ListTickets and Annotate operate on
// the in-process registry, so they work regardless of which backend is
// active.
package tickets

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/zeroroot-ai/gibson-redteam-example/findings-sink/internal/backend"
	pb "github.com/zeroroot-ai/gibson-redteam-example/findings-sink/api/gen/gibson/plugins/findingssink/v1"
)

// Store holds filed tickets and mirrors new ones to a backend.Sink. It
// is safe for concurrent use.
type Store struct {
	mu    sync.Mutex
	seq   int
	byID  map[string]*pb.Ticket
	order []string
	sink  backend.Sink
	now   func() int64 // injectable clock for tests
}

// NewStore returns an empty Store backed by sink.
func NewStore(sink backend.Sink) *Store {
	return &Store{
		byID: make(map[string]*pb.Ticket),
		sink: sink,
		now:  func() int64 { return time.Now().Unix() },
	}
}

// File records a finding as a new ticket and mirrors it to the backend.
// The ticket id is always returned (the in-process registry is
// authoritative); a non-nil error means only that the external mirror
// failed — the ticket is still filed locally.
func (s *Store) File(ctx context.Context, f *pb.Finding) (string, error) {
	s.mu.Lock()
	s.seq++
	id := "FS-" + strconv.Itoa(s.seq)
	t := &pb.Ticket{TicketId: id, Finding: f, FiledAt: s.now()}
	s.byID[id] = t
	s.order = append(s.order, id)
	sink := s.sink
	s.mu.Unlock()

	if err := sink.Save(ctx, t); err != nil {
		return id, fmt.Errorf("mirror to %s backend: %w", sink.Name(), err)
	}
	return id, nil
}

// List returns filed tickets in filing order, optionally filtered by
// severity and/or category. Empty filters match everything.
func (s *Store) List(severity, category string) []*pb.Ticket {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*pb.Ticket, 0, len(s.order))
	for _, id := range s.order {
		t := s.byID[id]
		if severity != "" && t.GetFinding().GetSeverity() != severity {
			continue
		}
		if category != "" && t.GetFinding().GetCategory() != category {
			continue
		}
		out = append(out, t)
	}
	return out
}

// Annotate appends a note to an existing ticket. It returns false if the
// ticket id is unknown.
func (s *Store) Annotate(id, note string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.byID[id]
	if !ok {
		return false
	}
	t.Notes = append(t.Notes, note)
	return true
}
