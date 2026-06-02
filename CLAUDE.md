# gibson-redteam-example — CLAUDE.md

> **Workflow rules:** see [`zeroroot-ai/.github` → `AGENTS.md`](https://github.com/zeroroot-ai/.github/blob/main/AGENTS.md) — canonical for branching / commits / PRs / releases / merging. Conventional Commits MANDATORY. Never push to main. Never force-push.

This is the per-repo addendum. Workspace-wide concerns live in the workspace `CLAUDE.md`; ADRs in [`zeroroot-ai/docs/adr/`](https://github.com/zeroroot-ai/docs/tree/main/adr).

## TL;DR

A complete, runnable **LLM-application red-team** built on the public [Gibson SDK](https://github.com/zeroroot-ai/sdk) — the canonical end-to-end example of how the three Gibson component archetypes fit together, and the platform's standing dev-cluster fixtures. Public OSS. Entry point: build/test each component independently (`cd <component> && make proto && make test && make build`).

## Architecture — the trio

Each component is its **own Go module** (no `go.work`, no `replace` — polyrepo discipline), wired together by the `llm-redteam-campaign` mission (a durable CUE definition under `missions/`):

| Directory | Kind | Role |
|---|---|---|
| `llm-redteam/` | **agent** | The orchestrating brain — two LLM slots, three-tier memory, GraphRAG, findings, plugin dispatch, budget-aware loop. Red-teams an `llm_chat` target. |
| `prompt-probe/` | **tool** | Stateless. Sends one crafted payload, returns a pattern-matched verdict, emits GraphRAG discoveries via proto field 100. Runs as a Redis work-queue worker. |
| `findings-sink/` | **plugin** | Stateful findings-tracker integration (`FileFinding`, `ListTickets`, `Annotate`). In-memory by default; a webhook backend activates when the `tracker-token` secret is present. |

This repo doubles as developer-observable dev-cluster fixtures — running them produces a real agent→tool→plugin round-trip with GraphRAG writes and submitted findings.

## Commands

Per component (each has its own `Makefile`):

```bash
cd prompt-probe && make proto && make test && make build
```

## Gotchas

- **Three independent modules, three independent build/test cycles.** There is no top-level build; never add `go.work` or `replace` to stitch them.
- **Proto sharing flows through BSR only** (ADR-0028). The tool and plugin publish their request/response messages to `buf.build/zeroroot-ai`; the agent consumes them as BSR deps. Generated bindings under each component's `api/gen/` are **gitignored** — run `make proto` to regenerate; do not commit them.
- Built against the **public OSS SDK only** — this is a customer-facing reference, so it must not reach for `platform-sdk` / `platform-clients` or any internal infra client.
- As a dev fixture it must keep producing an observable round-trip on the kind cluster; a component that can't come up is a bug, not an opt-out.

## Links

- Org-level workflow: [`AGENTS.md`](https://github.com/zeroroot-ai/.github/blob/main/AGENTS.md)
- Gibson SDK: [`github.com/zeroroot-ai/sdk`](https://github.com/zeroroot-ai/sdk)
- PR checklist: [`docs/agents/pr-checklist.md`](https://github.com/zeroroot-ai/docs/blob/main/agents/pr-checklist.md)
