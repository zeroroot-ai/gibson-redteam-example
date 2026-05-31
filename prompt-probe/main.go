// Command prompt-probe is a Gibson tool.
//
// See AGENTS.md for the full Gibson tool contract this binary implements,
// including the platform-wide rule that proto field 100 is reserved for
// gibson.graphrag.v1.DiscoveryResult on every tool response message.
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/zeroroot-ai/sdk/graphrag"
	"github.com/zeroroot-ai/sdk/serve"
	"github.com/zeroroot-ai/sdk/types"
	"google.golang.org/protobuf/proto"

	pb "github.com/zeroroot-ai/gibson-redteam-example/prompt-probe/api/gen/gibson/tools/promptprobe/v1"
	"github.com/zeroroot-ai/gibson-redteam-example/prompt-probe/gen"
)

// PromptProbeTool is the tool implementation.
type PromptProbeTool struct{}

func (t *PromptProbeTool) Name() string              { return "prompt-probe" }
func (t *PromptProbeTool) Version() string           { return "0.1.0" }
func (t *PromptProbeTool) Description() string       { return "prompt-probe tool" }
func (t *PromptProbeTool) Tags() []string            { return []string{"llm", "redteam", "probe"} }
func (t *PromptProbeTool) InputMessageType() string  { return "gibson.tools.promptprobe.v1.PromptProbeRequest" }
func (t *PromptProbeTool) OutputMessageType() string { return "gibson.tools.promptprobe.v1.PromptProbeResponse" }

// OntologyExtension implements the optional serve.OntologyContributor
// interface. The SDK's serve.Tool runtime type-asserts against it at
// enrollment and forwards the result to the daemon's reasoner. The
// gen.OntologyExtension() function is byte-stable and regenerated from
// ontology.yaml via `gibson component generate`.
//
// If your tool has no ontology to contribute, leaving the prefixes /
// hierarchies / equivalences / ifps blocks empty in ontology.yaml makes
// this method a harmless no-op — the SDK skips an empty extension on the
// wire.
func (t *PromptProbeTool) OntologyExtension() graphrag.OntologyExtension {
	return gen.OntologyExtension()
}

// ExecuteProto is the tool's entrypoint. The daemon serialises the agent's
// request into the input proto.Message and unwraps the response.
//
// IMPORTANT: populate the Discovery field (proto field 100) with any
// entities + relationships your tool discovered. The Gibson daemon's
// DiscoveryProcessor reflects on field 100 of every tool response and
// writes the entries into the GraphRAG (Neo4j) knowledge graph
// automatically. See AGENTS.md and core/sdk/api/proto/gibson/graphrag/v1/.
func (t *PromptProbeTool) ExecuteProto(ctx context.Context, in proto.Message) (proto.Message, error) {
	_ = in.(*pb.PromptProbeRequest) // type-assert; replace with real fields

	// TODO: implement the tool's real behaviour and populate Discovery
	// with whatever Hosts / Ports / Services / etc. it learned.

	return &pb.PromptProbeResponse{
		// Discovery: &graphragpb.DiscoveryResult{...},
	}, nil
}

// Health is the tool's readiness probe. The SDK's serve.Tool runtime
// calls it for the daemon's health checks.
func (t *PromptProbeTool) Health(ctx context.Context) types.HealthStatus {
	return types.HealthStatus{Status: types.StatusHealthy}
}

func main() {
	if err := serve.Tool(&PromptProbeTool{}); err != nil {
		slog.Error("serve tool", "err", err)
		os.Exit(1)
	}
}
