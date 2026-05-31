package tickets_test

import (
	"context"
	"errors"
	"testing"

	"github.com/zeroroot-ai/gibson-redteam-example/findings-sink/internal/tickets"
	pb "github.com/zeroroot-ai/gibson-redteam-example/findings-sink/api/gen/gibson/plugins/findingssink/v1"
)

// fakeSink records saved tickets and can be made to fail.
type fakeSink struct {
	saved []*pb.Ticket
	err   error
}

func (f *fakeSink) Save(_ context.Context, t *pb.Ticket) error {
	if f.err != nil {
		return f.err
	}
	f.saved = append(f.saved, t)
	return nil
}
func (f *fakeSink) Name() string { return "fake" }

func TestFileMirrorsAndAssignsIDs(t *testing.T) {
	sink := &fakeSink{}
	s := tickets.NewStore(sink)

	id1, err := s.File(context.Background(), &pb.Finding{Title: "a", Severity: "high"})
	if err != nil {
		t.Fatalf("File: %v", err)
	}
	id2, _ := s.File(context.Background(), &pb.Finding{Title: "b", Severity: "low"})
	if id1 == id2 {
		t.Errorf("ids not unique: %s == %s", id1, id2)
	}
	if len(sink.saved) != 2 {
		t.Errorf("backend got %d tickets, want 2", len(sink.saved))
	}
	if sink.saved[0].GetFiledAt() == 0 {
		t.Errorf("filed_at not set")
	}
}

func TestFileReturnsIDEvenWhenMirrorFails(t *testing.T) {
	sink := &fakeSink{err: errors.New("boom")}
	s := tickets.NewStore(sink)

	id, err := s.File(context.Background(), &pb.Finding{Title: "a"})
	if id == "" {
		t.Errorf("expected a ticket id even when mirror fails")
	}
	if err == nil {
		t.Errorf("expected the mirror error surfaced")
	}
	// Ticket must still be locally listable.
	if len(s.List("", "")) != 1 {
		t.Errorf("ticket not stored locally despite mirror failure")
	}
}

func TestListFilters(t *testing.T) {
	s := tickets.NewStore(&fakeSink{})
	s.File(context.Background(), &pb.Finding{Title: "x", Severity: "high", Category: "jailbreak"})
	s.File(context.Background(), &pb.Finding{Title: "y", Severity: "low", Category: "prompt_injection"})
	s.File(context.Background(), &pb.Finding{Title: "z", Severity: "high", Category: "prompt_injection"})

	if got := len(s.List("", "")); got != 3 {
		t.Errorf("no filter → %d, want 3", got)
	}
	if got := len(s.List("high", "")); got != 2 {
		t.Errorf("severity=high → %d, want 2", got)
	}
	if got := len(s.List("", "prompt_injection")); got != 2 {
		t.Errorf("category=prompt_injection → %d, want 2", got)
	}
	if got := len(s.List("high", "prompt_injection")); got != 1 {
		t.Errorf("both filters → %d, want 1", got)
	}
}

func TestAnnotate(t *testing.T) {
	s := tickets.NewStore(&fakeSink{})
	id, _ := s.File(context.Background(), &pb.Finding{Title: "x"})

	if !s.Annotate(id, "looked into it") {
		t.Errorf("Annotate(existing) = false, want true")
	}
	if s.Annotate("FS-999", "nope") {
		t.Errorf("Annotate(missing) = true, want false")
	}
	notes := s.List("", "")[0].GetNotes()
	if len(notes) != 1 || notes[0] != "looked into it" {
		t.Errorf("notes = %v", notes)
	}
}
