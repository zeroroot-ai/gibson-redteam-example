# findings-sink

A Gibson plugin scaffolded by `gibson component init`.

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
