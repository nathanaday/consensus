# MCP Project Conventions

These conventions govern the Consensus MCP server (`mcp/`).

## Module

- Module path: `github.com/nathanaday/consensus/mcp`.
- Server identity: `Name: "consensus"`.

## Tools

- One tool per file in `internal/tools/`. The filename equals the tool name
  (e.g. `greet.go`, later `extract_csv.go`).
- Each tool file owns its `Input` / `Output` types and its handler function.

## Registration seam

- All tools are registered through `tools.Register(server *mcp.Server)` in
  `internal/tools/register.go`.
- `main.go` never calls `mcp.AddTool` directly and does not change when tools
  are added.

## Entry point

- `cmd/server/main.go` handles process lifecycle only: construct the server,
  call `tools.Register`, and run the transport. No business logic.
