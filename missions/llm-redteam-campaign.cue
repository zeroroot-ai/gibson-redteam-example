// llm-redteam-campaign — red-team an LLM chat endpoint.
//
// A single-node mission: the llm-redteam agent probes the target with
// prompt-probe, classifies each verdict, and files findings through
// findings-sink. Durable, run-any-time.
//
// Override before submitting:
//   targetRef: "<your-llm_chat-target-name-or-id>"
//
// Submit with:
//   gibson mission submit missions/llm-redteam-campaign.cue --target <ref>

import missionv1 "github.com/zeroroot-ai/sdk/api/proto/gibson/mission/v1"

mission: missionv1.#MissionDefinition & {
	name:        "llm-redteam-campaign"
	description: "Red-team an LLM chat endpoint with prompt-probe + llm-redteam."
	version:     "1.0.0"
	targetRef:   ""

	nodes: {
		redteam: {
			id:   "redteam"
			type: missionv1.#NODE_TYPE_AGENT
			agentConfig: {
				agentName: "llm-redteam"
			}
		}
	}
	edges: []
	entryPoints: ["redteam"]
	exitPoints: ["redteam"]
}
