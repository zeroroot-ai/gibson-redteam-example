// Command prompt-probe is a Gibson tool.
//
// prompt-probe sends one crafted payload to an LLM chat/completion
// endpoint and returns a structured, pattern-matched verdict plus a
// GraphRAG discovery describing what was exercised. It is stateless and
// does no LLM reasoning — the calling agent decides what the verdict
// means.
//
// See AGENTS.md for the full Gibson tool contract, including the
// platform-wide rule that proto field 100 is reserved for
// gibson.graphrag.v1.DiscoveryResult on every tool response message.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/zeroroot-ai/sdk/graphrag"
	"github.com/zeroroot-ai/sdk/serve"
	"github.com/zeroroot-ai/sdk/types"
	"google.golang.org/protobuf/proto"

	pb "github.com/zeroroot-ai/gibson-redteam-example/prompt-probe/api/gen/gibson/tools/promptprobe/v1"
	"github.com/zeroroot-ai/gibson-redteam-example/prompt-probe/gen"
	"github.com/zeroroot-ai/gibson-redteam-example/prompt-probe/internal/discovery"
	"github.com/zeroroot-ai/gibson-redteam-example/prompt-probe/internal/signalmatch"
)

// maxBodyBytes caps how much of the target's response body we read and
// return, to keep responses bounded.
const maxBodyBytes = 64 << 10

// defaultPromptField is the JSON body field the payload is placed in
// when the request does not specify one.
const defaultPromptField = "prompt"

// PromptProbeTool is the tool implementation.
type PromptProbeTool struct{}

func (t *PromptProbeTool) Name() string        { return "prompt-probe" }
func (t *PromptProbeTool) Version() string     { return "0.1.0" }
func (t *PromptProbeTool) Description() string {
	return "Sends one crafted payload to an LLM endpoint and returns a pattern-matched verdict"
}
func (t *PromptProbeTool) Tags() []string            { return []string{"llm", "redteam", "probe"} }
func (t *PromptProbeTool) InputMessageType() string  { return "gibson.tools.promptprobe.v1.PromptProbeRequest" }
func (t *PromptProbeTool) OutputMessageType() string { return "gibson.tools.promptprobe.v1.PromptProbeResponse" }

// OntologyExtension implements the optional serve.OntologyContributor
// interface. The SDK's serve.Tool runtime type-asserts against it at
// enrollment and forwards the result to the daemon's reasoner.
func (t *PromptProbeTool) OntologyExtension() graphrag.OntologyExtension {
	return gen.OntologyExtension()
}

// ExecuteProto is the tool's entrypoint. The daemon serialises the
// agent's request into the input proto.Message and unwraps the response.
func (t *PromptProbeTool) ExecuteProto(ctx context.Context, in proto.Message) (proto.Message, error) {
	req, ok := in.(*pb.PromptProbeRequest)
	if !ok {
		return nil, fmt.Errorf("prompt-probe: expected *PromptProbeRequest, got %T", in)
	}
	return runProbe(ctx, http.DefaultClient, req)
}

// runProbe performs the HTTP probe and assembles the response. It is
// separated from ExecuteProto so it can be driven by an httptest server
// in tests. A transport error is not fatal — it is reported as a
// status-0 response with the error in the body, which is itself a useful
// signal to the calling agent.
func runProbe(ctx context.Context, client *http.Client, req *pb.PromptProbeRequest) (*pb.PromptProbeResponse, error) {
	field := req.GetPromptField()
	if field == "" {
		field = defaultPromptField
	}
	bodyJSON, err := json.Marshal(map[string]string{field: req.GetPayload()})
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, req.GetTargetUrl(), bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range req.GetHeaders() {
		httpReq.Header.Set(k, v)
	}

	start := time.Now()
	resp, err := client.Do(httpReq)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		// Transport failure: report as a status-0 verdict, still recording
		// the endpoint in the discovery graph.
		return &pb.PromptProbeResponse{
			StatusCode: 0,
			Body:       err.Error(),
			LatencyMs:  latency,
			Discovery:  discovery.Build(req.GetTargetUrl(), http.MethodPost, 0, req.GetTechniqueId()),
		}, nil
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	body := string(raw)

	refused, signals := signalmatch.Analyze(resp.StatusCode, body)

	return &pb.PromptProbeResponse{
		StatusCode:    int32(resp.StatusCode),
		Body:          body,
		LatencyMs:     latency,
		Refused:       refused,
		SignalMatches: toProtoSignals(signals),
		Discovery:     discovery.Build(req.GetTargetUrl(), http.MethodPost, int32(resp.StatusCode), req.GetTechniqueId()),
	}, nil
}

// toProtoSignals converts analyzer signals to their proto form.
func toProtoSignals(signals []signalmatch.Signal) []*pb.SignalMatch {
	if len(signals) == 0 {
		return nil
	}
	out := make([]*pb.SignalMatch, 0, len(signals))
	for _, s := range signals {
		out = append(out, &pb.SignalMatch{Kind: s.Kind, Pattern: s.Pattern})
	}
	return out
}

// Health is the tool's readiness probe.
func (t *PromptProbeTool) Health(ctx context.Context) types.HealthStatus {
	return types.HealthStatus{Status: types.StatusHealthy}
}

func main() {
	if err := serve.Tool(&PromptProbeTool{}); err != nil {
		slog.Error("serve tool", "err", err)
		os.Exit(1)
	}
}
