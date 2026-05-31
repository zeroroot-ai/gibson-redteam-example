# Adding a method to prompt-probe

A "tool method" in Gibson is the proto request/response pair this tool
serves. Most tools have a single method (one Request, one Response);
multi-method tools are unusual but supported by adding additional
message types and dispatching inside `ExecuteProto`.

## Step 1 — extend the proto

Edit `api/proto/gibson/tools/promptprobe/v1/promptprobe.proto`. Add fields to
`PromptProbeRequest` and `PromptProbeResponse`. **Do not change field 100
on any response message** — that's reserved platform-wide for
`gibson.graphrag.v1.DiscoveryResult`.

Example:

```proto
message PromptProbeRequest {
  string target = 1;
  int32  timeout_seconds = 2;       // new field
  repeated string options = 3;       // new field
}

message PromptProbeResponse {
  string raw_output = 1;
  int32  exit_code  = 2;             // new field
  // ── Field 100 reserved ──
  gibson.graphrag.v1.DiscoveryResult discovery = 100;
}
```

## Step 2 — regenerate Go bindings

```sh
make proto
```

`buf generate` produces `api/gen/gibson/tools/promptprobe/v1/promptprobe.pb.go`. The
generated `*Request` / `*Response` Go structs gain the new fields.

## Step 3 — implement

Edit `main.go`'s `ExecuteProto`:

```go
func (t *PromptProbeTool) ExecuteProto(ctx context.Context, in proto.Message) (proto.Message, error) {
    req := in.(*pb.PromptProbeRequest)

    // Use req.Target, req.TimeoutSeconds, req.Options
    out, err := doWork(ctx, req)
    if err != nil {
        return nil, err
    }

    discovery := buildDiscovery(out)  // see prompts/add-discovery.md

    return &pb.PromptProbeResponse{
        RawOutput: string(out),
        ExitCode:  0,
        Discovery: discovery,
    }, nil
}
```

## Step 4 — re-register

If the proto's package path changes, the daemon's registered
descriptor cache for this tool must be refreshed. Restart the tool
binary; `serve.Tool` re-registers the FileDescriptorSet on first
connection.

## Step 5 — agent calling site

Agents import this tool's generated Go bindings and invoke via the
harness:

```go
req := &promptprobepb.PromptProbeRequest{Target: "example.com"}
resp := &promptprobepb.PromptProbeResponse{}
if err := h.CallToolProto(ctx, "prompt-probe", req, resp); err != nil {
    return err
}
```

## Don't

- Don't break field numbers — `buf` will catch breaking changes via
  `buf breaking`. Removed or renumbered fields require coordination
  with every agent that imports this tool's bindings.
- Don't rename the proto `package` declaration without updating every
  consumer — the FQ message names (`gibson.tools.promptprobe.v1.*`) are
  the daemon's dispatch keys.
