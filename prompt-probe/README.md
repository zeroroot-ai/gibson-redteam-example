# prompt-probe

The **tool** of the red-team reference trio. Stateless: given one crafted
payload and an LLM endpoint, it sends a single probe and returns a
structured, pattern-matched verdict. It performs no LLM reasoning — the
`llm-redteam` agent decides what the verdict means.

## What it does

`PromptProbeRequest{ target_url, payload, technique_id, headers, prompt_field }`
→ HTTP POST `{ "<prompt_field|prompt>": payload }` → `PromptProbeResponse`:

| Field | Meaning |
|-------|---------|
| `status_code`, `body`, `latency_ms` | the raw exchange |
| `refused` | the response looks like a model refusal |
| `signal_matches[]` | `refusal` / `leak` / `injection_success` indicators that fired |
| `discovery` (field 100) | the probed Endpoint + the Technique it was `TESTED_WITH`, auto-written to GraphRAG |

A transport failure is reported as a `status_code: 0` verdict (still
recording the endpoint), not an error — a connection failure is itself a
signal to the agent.

### Deep modules (unit-tested)
- `internal/signalmatch` — pure response → `(refused, []Signal)` classifier
- `internal/discovery` — pure probe result → `DiscoveryResult` mapping

See **[AGENTS.md](./AGENTS.md)** for the Gibson tool contract — including
the platform rule that **proto field 100 on every tool response message
is reserved for `gibson.graphrag.v1.DiscoveryResult`**, which the daemon
auto-extracts into the GraphRAG knowledge graph. What follows is the
five-command quickstart.

## Quickstart

```sh
# 1. Generate Go bindings from api/proto/gibson/tools/promptprobe/v1/promptprobe.proto
make proto

# 2. Build
make build

# 3. Register (paste the enroll_command's flags from the dashboard)
gibson component register \
  --client-id <id> \
  --client-secret - \
  --gibson-url <url>

# 4. Run (reads creds from ~/.gibson/tool/credentials)
make run
```

## Container

```sh
make image
docker run --rm \
  -e GIBSON_URL=https://api.zeroroot.ai \
  prompt-probe:0.1.0
```

## Proto layout

```
api/proto/gibson/tools/promptprobe/v1/promptprobe.proto   # your tool's request/response
api/gen/                                                     # generated Go (gitignored)
proto/vendor/gibson/graphrag/v1/            # vendored DiscoveryResult
proto/vendor/taxonomy/v1/                   # vendored taxonomy enums
buf.yaml, buf.gen.yaml                      # buf v2 configuration
```
