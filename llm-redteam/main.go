// Command llm-redteam is the orchestrating agent of the red-team
// reference trio. Given an llm_chat target, it probes the endpoint with
// the prompt-probe tool, classifies each verdict, submits findings, and
// files them through the findings-sink plugin — tracking budget and
// emitting dashboard-visible traces.
//
// See AGENTS.md for the full Gibson agent contract.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	sdk "github.com/zeroroot-ai/sdk"
	"github.com/zeroroot-ai/sdk/agent"
	"github.com/zeroroot-ai/sdk/llm"
	"go.opentelemetry.io/otel/attribute"

	graphragpb "github.com/zeroroot-ai/sdk/api/gen/gibson/graphrag/v1"
	"github.com/zeroroot-ai/sdk/types"

	"github.com/zeroroot-ai/gibson-redteam-example/llm-redteam/internal/budget"
	"github.com/zeroroot-ai/gibson-redteam-example/llm-redteam/internal/campaign"
)

func main() {
	a, err := sdk.NewAgent(
		sdk.WithName("llm-redteam"),
		sdk.WithVersion("0.1.0"),
		sdk.WithDescription("LLM-application red-team agent"),
		// primary: the reasoning slot — needs tool use to drive prompt-probe.
		sdk.WithLLMSlot("primary", llm.SlotRequirements{
			MinContextWindow: 16000,
			RequiredFeatures: []string{"function_calling"},
		}),
		// fast: a cheaper slot for quick classification (JSON mode).
		sdk.WithLLMSlot("fast", llm.SlotRequirements{
			MinContextWindow: 4000,
			RequiredFeatures: []string{"json_mode"},
		}),
		sdk.WithExecuteFunc(execute),
	)
	if err != nil {
		slog.Error("create agent", "err", err)
		os.Exit(1)
	}

	if err := sdk.ServeAgent(a); err != nil {
		slog.Error("serve agent", "err", err)
		os.Exit(1)
	}
}

// execute runs one red-team campaign against the mission's target.
func execute(ctx context.Context, h agent.Harness, task agent.Task) (agent.Result, error) {
	log := h.Logger()

	// Emit a span carrying the Langfuse trace contract the dashboard
	// Traces viewer filters on (user id + tags).
	ctx, span := h.Tracer().Start(ctx, "llm-redteam.campaign")
	defer span.End()
	span.SetAttributes(
		attribute.String("langfuse.user.id", h.Mission().Name),
		attribute.StringSlice("langfuse.trace.tags", []string{"llm-redteam", "reference-trio"}),
	)

	target := targetURL(h.Target())
	if target == "" {
		return agent.Result{
			Status: agent.StatusFailed,
			Output: "no target URL found in target.connection (expected url/endpoint/base_url)",
		}, nil
	}

	// Working memory: record what this run is operating on.
	if err := h.Memory().Working().Set(ctx, "target", target); err != nil {
		log.WarnContext(ctx, "working memory set failed", "err", err)
	}

	// Best-effort GraphRAG recall: how many techniques have prior runs
	// already exercised against this graph? Purely informational here.
	if prior := recallTechniques(ctx, h); prior >= 0 {
		log.InfoContext(ctx, "prior techniques in graph", "count", prior)
		span.SetAttributes(attribute.Int("graph.prior_techniques", prior))
	}

	limits := budget.Limits{
		MaxTurns:  task.Constraints.MaxTurns,
		MaxTokens: task.Constraints.MaxTokens,
	}

	out := campaign.Run(ctx, h, target, campaign.DefaultCandidates(), limits)
	span.SetAttributes(
		attribute.Int("probes.run", out.ProbesRun),
		attribute.Int("findings.filed", out.FindingsFiled),
		attribute.String("campaign.status", string(out.Status)),
	)

	return agent.Result{
		Status: mapStatus(out.Status),
		Output: fmt.Sprintf("campaign %s: ran %d probes, filed %d findings", out.Status, out.ProbesRun, out.FindingsFiled),
		Metadata: map[string]any{
			"probes_run":      out.ProbesRun,
			"findings_filed":  out.FindingsFiled,
			"campaign_status": string(out.Status),
		},
	}, nil
}

// targetURL extracts the endpoint from a target's connection map,
// tolerating the common key spellings.
func targetURL(t types.TargetInfo) string {
	for _, k := range []string{"url", "endpoint", "base_url"} {
		if v, ok := t.Connection[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

// recallTechniques returns how many Technique nodes exist in the
// knowledge graph for context, or -1 if the query fails. Best-effort.
func recallTechniques(ctx context.Context, h agent.Harness) int {
	res, err := h.QueryNodes(ctx, &graphragpb.GraphQuery{
		NodeTypes: []string{"Technique"},
		TopK:      50,
	})
	if err != nil {
		return -1
	}
	return len(res)
}

// mapStatus maps the campaign's budget status to an agent result status.
// The SDK has no dedicated budget-exceeded status, so it folds into
// partial (the metadata records the precise campaign_status).
func mapStatus(s budget.Status) agent.ResultStatus {
	if s == budget.StatusSuccess {
		return agent.StatusSuccess
	}
	return agent.StatusPartial
}
