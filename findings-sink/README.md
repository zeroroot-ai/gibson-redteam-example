# findings-sink

The **plugin** of the red-team reference trio: a stateful findings
tracker the `llm-redteam` agent files confirmed findings into.

## Methods

| Method | Request → Response | Behavior |
|--------|--------------------|----------|
| `FileFinding` | `Finding` → `{ ticket_id }` | mints a ticket id, stores it, mirrors it to the backend |
| `ListTickets` | `{ severity?, category? }` → `[]Ticket` | lists filed tickets (in-process), filterable |
| `Annotate` | `{ ticket_id, note }` → `{ ok }` | appends a note to a ticket |

## Backends & the secret

The plugin keeps an **authoritative in-process ticket registry**, so
`ListTickets`/`Annotate` always work. A backend `Sink` decides where a
*newly filed* ticket is also pushed:

- **in-memory** (default) — zero dependencies; nothing external.
- **webhook** — POSTs each filed ticket to `FINDINGS_SINK_WEBHOOK_URL`,
  authenticated with the `cred:tracker-token` secret declared in
  `plugin.yaml`.

`plugin.yaml` declares `cred:tracker-token` as `required: false` so the
plugin starts dependency-free; declaring it still places the secret
under the daemon's lifecycle (rotation/invalidation) and surfaces it on
the dashboard Secrets page.

> Note: the current SDK surfaces a declared secret to a plugin only via
> Serve's startup pre-resolution and rotation events — there is no
> handler-accessible resolve API. This example therefore reads the
> webhook bearer token from the environment
> (`FINDINGS_SINK_TRACKER_TOKEN`) as the projection mechanism. Tracked
> upstream (see the repo's epic).

### Deep modules (unit-tested)
- `internal/tickets` — the concurrent ticket registry (File/List/Annotate)
- `internal/backend` — the `Sink` interface, in-memory + webhook impls, and `Select`

See **[AGENTS.md](./AGENTS.md)** for the full Gibson plugin contract —
manifest schema, lifecycle states, secrets-broker rules, exit-75
rotation, and the SDK source paths to grep. What follows is the
four-command quickstart.

## Quickstart

```sh
# 1. Generate Go bindings from gibson/plugins/findingssink/v1/findingssink.proto
make proto

# 2. Build
make build

# 3. Register (paste the bootstrap-token from the dashboard)
gibson component register --token <bootstrap-token>

# 4. Run (reads ~/.gibson/plugin/findings-sink/host_key)
make run
```

## Container (pod runtime mode)

```sh
docker build -t findings-sink:0.1.0 .
docker run --rm \
  -e GIBSON_URL=https://api.zeroroot.ai \
  -e GIBSON_PLUGIN_RUNTIME=pod \
  findings-sink:0.1.0
```

## Operator runbooks (internal)

- `enterprise/deploy/docs/runbooks/plugin-runtime.md`
- `enterprise/deploy/docs/runbooks/secrets-broker.md`
