# gibson-redteam-example

A complete, runnable **LLM-application red-team** built on the [Gibson SDK](https://github.com/zeroroot-ai/sdk) — the canonical end-to-end example of how the three Gibson component archetypes fit together, and the platform's standing dev-cluster fixtures.

Clone this repo to learn how an **agent**, a **tool**, and a **plugin** collaborate on a real mission: an LLM-driven agent reasons about a chat target, calls a stateless probe tool, writes what it discovers into the GraphRAG knowledge graph, submits findings, and files them through a stateful integration plugin.

## The trio

| Directory | Kind | What it does |
|-----------|------|--------------|
| [`llm-redteam/`](./llm-redteam) | **agent** | The orchestrating brain. Two LLM slots, three-tier memory, GraphRAG, findings, plugin dispatch, budget-aware loop. Red-teams an `llm_chat` target. |
| [`prompt-probe/`](./prompt-probe) | **tool** | Stateless. Sends one crafted payload to the target and returns a structured, pattern-matched verdict. Emits GraphRAG discoveries via proto field 100. Runs as a Redis work-queue worker. |
| [`findings-sink/`](./findings-sink) | **plugin** | Stateful "findings tracker" integration with three methods (`FileFinding`, `ListTickets`, `Annotate`). In-memory backend by default; a webhook backend activates when the `tracker-token` secret is present. |

The mission that wires them together is `llm-redteam-campaign` (a durable CUE definition; see [`llm-redteam/`](./llm-redteam)).

## Layout

Each component is its **own Go module** (no `go.work`, no `replace` — polyrepo discipline). Build and test them independently:

```sh
cd prompt-probe && make proto && make test && make build
```

Proto sharing between components flows through [BSR](https://buf.build) only — the tool and plugin publish their request/response messages to `buf.build/zeroroot-ai`, and the agent consumes them as BSR dependencies (ADR-0028). Generated Go bindings under `api/gen/` are gitignored; run `make proto` to regenerate.

## Status

This repo is built wave-by-wave under epic **redteam-reference-trio** (`zeroroot-ai/.github#164`). Each component starts as a scaffolded stub and is filled in by a tracked issue. See the epic board for progress.

## License

Business Source License 1.1 — see [LICENSE](./LICENSE).
