# llm-redteam

The **agent** of the red-team reference trio — the orchestrating brain.
Given an `llm_chat` target it runs a budget-bounded campaign:

1. emits a trace span with the Langfuse contract (`langfuse.user.id`,
   `langfuse.trace.tags`) so the run is filterable in the dashboard
2. records the target in **working memory** and recalls prior `Technique`
   nodes from **GraphRAG**
3. for each candidate payload: calls the **`prompt-probe`** tool
   (`CallToolProto`), classifies the verdict, and on a hit submits a
   **finding** and files it through the **`findings-sink`** plugin
   (`QueryPlugin`)
4. honors `task.Constraints` (max turns/tokens) and returns
   success / partial per the budget outcome

### LLM slots
- `primary` — reasoning, requires `function_calling`
- `fast` — classification, requires `json_mode`

### Deep modules (unit-tested)
- `internal/campaign` — the loop (driven by a narrow harness interface; fake-harness tested)
- `internal/classify` — verdict → finding category/severity
- `internal/budget` — turn/token limits → terminal status

### Proto consumption (ADR-0028)
The agent calls the tool via its **BSR-published** proto, imported as the
BSR-generated Go SDK `buf.build/gen/go/zeroroot-ai/prompt-probe/...` — no
local proto includes. The plugin is reached through `QueryPlugin` with a
`map[string]any`, so it needs no generated plugin types.

See **[AGENTS.md](./AGENTS.md)** for the full Gibson agent contract this
scaffold implements (LLM slots, harness API, lifecycle, do-not-do
list). What follows is the four-command quickstart.

## Quickstart

```sh
# 1. Build
make build

# 2. Register (paste the enroll_command's flags from the dashboard)
gibson component register \
  --client-id <id> \
  --client-secret - \
  --gibson-url <url>

# 3. Run (reads creds from ~/.gibson/agent/credentials)
make run
```

## Container

```sh
make image
docker run --rm \
  -e GIBSON_URL=https://api.zeroroot.ai \
  llm-redteam:0.1.0
```
