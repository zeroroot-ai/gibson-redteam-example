package campaign_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/zeroroot-ai/sdk/finding"
	"google.golang.org/protobuf/proto"

	promptprobev1 "buf.build/gen/go/zeroroot-ai/prompt-probe/protocolbuffers/go/gibson/tools/promptprobe/v1"

	"github.com/zeroroot-ai/gibson-redteam-example/llm-redteam/internal/budget"
	"github.com/zeroroot-ai/gibson-redteam-example/llm-redteam/internal/campaign"
)

type pluginCall struct {
	method string
	params map[string]any
}

// fakeHarness implements campaign.Harness. It returns a programmed probe
// response keyed by technique id and records finding/plugin activity.
type fakeHarness struct {
	responses   map[string]*promptprobev1.PromptProbeResponse
	findings    []*finding.Finding
	pluginCalls []pluginCall
}

func (f *fakeHarness) CallToolProto(_ context.Context, _ string, req, resp proto.Message) error {
	in := req.(*promptprobev1.PromptProbeRequest)
	out := resp.(*promptprobev1.PromptProbeResponse)
	if r, ok := f.responses[in.GetTechniqueId()]; ok {
		proto.Merge(out, r)
	}
	return nil
}

func (f *fakeHarness) SubmitFinding(_ context.Context, fnd *finding.Finding) error {
	f.findings = append(f.findings, fnd)
	return nil
}

func (f *fakeHarness) QueryPlugin(_ context.Context, _, method string, params map[string]any) (any, error) {
	f.pluginCalls = append(f.pluginCalls, pluginCall{method: method, params: params})
	return map[string]any{"ticket_id": "FS-1"}, nil
}

func (f *fakeHarness) Logger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func injection(tech string) *promptprobev1.PromptProbeResponse {
	return &promptprobev1.PromptProbeResponse{
		StatusCode:    200,
		Body:          "INJECTION-OK",
		SignalMatches: []*promptprobev1.SignalMatch{{Kind: "injection_success", Pattern: "injection-ok"}},
	}
}

func refusal(tech string) *promptprobev1.PromptProbeResponse {
	return &promptprobev1.PromptProbeResponse{
		StatusCode:    200,
		Body:          "I'm sorry, I can't help with that.",
		Refused:       true,
		SignalMatches: []*promptprobev1.SignalMatch{{Kind: "refusal", Pattern: "i can't"}},
	}
}

func TestRun_filesFindingsForHits(t *testing.T) {
	h := &fakeHarness{responses: map[string]*promptprobev1.PromptProbeResponse{
		"AML.T0051": injection("AML.T0051"), // hit
		"AML.T0054": refusal("AML.T0054"),   // defended
	}}
	candidates := []campaign.Candidate{
		{Payload: "x", TechniqueID: "AML.T0051"},
		{Payload: "y", TechniqueID: "AML.T0054"},
	}

	out := campaign.Run(context.Background(), h, "https://llm.example", candidates, budget.Limits{})

	if out.ProbesRun != 2 {
		t.Errorf("ProbesRun = %d, want 2", out.ProbesRun)
	}
	if out.FindingsFiled != 1 {
		t.Errorf("FindingsFiled = %d, want 1 (injection only)", out.FindingsFiled)
	}
	if out.Status != budget.StatusSuccess {
		t.Errorf("Status = %q, want success", out.Status)
	}
	if len(h.findings) != 1 || h.findings[0].Category != "prompt_injection" {
		t.Errorf("expected one prompt_injection finding, got %+v", h.findings)
	}
	if len(h.pluginCalls) != 1 || h.pluginCalls[0].method != "FileFinding" {
		t.Errorf("expected one FileFinding plugin call, got %+v", h.pluginCalls)
	}
	// The plugin call must carry the finding payload.
	fp, _ := h.pluginCalls[0].params["finding"].(map[string]any)
	if fp == nil || fp["category"] != "prompt_injection" {
		t.Errorf("plugin finding params malformed: %+v", h.pluginCalls[0].params)
	}
}

func TestRun_budgetStopsEarly(t *testing.T) {
	h := &fakeHarness{responses: map[string]*promptprobev1.PromptProbeResponse{
		"AML.T0051": injection("AML.T0051"),
		"AML.T0054": injection("AML.T0054"),
	}}
	candidates := []campaign.Candidate{
		{Payload: "x", TechniqueID: "AML.T0051"},
		{Payload: "y", TechniqueID: "AML.T0054"},
	}

	out := campaign.Run(context.Background(), h, "https://llm.example", candidates, budget.Limits{MaxTurns: 1})

	if out.ProbesRun != 1 {
		t.Errorf("ProbesRun = %d, want 1 under MaxTurns=1", out.ProbesRun)
	}
	if out.Status != budget.StatusPartial {
		t.Errorf("Status = %q, want partial", out.Status)
	}
}

func TestDefaultCandidates(t *testing.T) {
	c := campaign.DefaultCandidates()
	if len(c) < 3 {
		t.Fatalf("want at least 3 default candidates, got %d", len(c))
	}
	for i, cand := range c {
		if cand.Payload == "" || cand.TechniqueID == "" {
			t.Errorf("candidate %d incomplete: %+v", i, cand)
		}
	}
}
