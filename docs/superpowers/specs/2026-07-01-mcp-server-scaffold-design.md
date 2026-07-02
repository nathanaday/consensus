# Consensus — Phase 1 MCP Server Scaffold

Date: 2026-07-01
Status: Approved (design)

## Goal

Stand up a "hello world" MCP server for the Consensus project using the Go MCP
SDK (`github.com/modelcontextprotocol/go-sdk`), laid out so that Phase 1 tools
(time-series data extraction and statistics) drop in without restructuring.

This session's deliverable is the scaffold and a working MCP handshake — not the
Phase 1 tools themselves.

## Project concept (from README)

Consensus is an MCP server for natural-language interaction with time-series /
IoT data streams: connect streams, analyze them, and attach context to data
points for situational awareness.

### Phase 1 scope (later specs, not this session)

- Ingest time-series data from memory, CSV, or relational databases.
- Native integration with Postgres and SQLite.
- Common time-series statistical operations.

## Repository shape

Consensus is a monorepo spanning two surfaces, established now:

```
consensus/
├── mcp/                     # Go MCP server — all Go code lives here
│   ├── go.mod               # module github.com/nathanaday/consensus/mcp
│   ├── README.md            # moved from repo root (git mv, user-owned)
│   ├── CONVENTIONS.md       # MCP project preferences (see below)
│   ├── cmd/
│   │   └── server/
│   │       └── main.go      # process lifecycle only
│   └── internal/
│       └── tools/
│           └── greet.go     # placeholder tool; one file per tool
└── app/                     # future full-stack app (undefined)
    └── README.md            # one-line stub marking future scope
```

- Go: 1.24.1 (installed).
- Module path: `github.com/nathanaday/consensus/mcp` (includes the subdir so
  imports and `go install` resolve correctly with go.mod under `mcp/`).

## Components

### `cmd/server/main.go`
Owns process lifecycle only: construct the `mcp.Server`
(`Name: "consensus"`), call `tools.Register(server)`, and run the
`StdioTransport` until the client disconnects. No business logic.

### `internal/tools`
Home for all MCP tools. Ships with `greet.go`, the SDK's `SayHi` example moved
into this package and exported. Exposes `Register(server *mcp.Server)` which
adds every tool. Phase 1 tools are added there and in their own files.

## MCP project preferences

Recorded durably in `mcp/CONVENTIONS.md`:

- **One tool per file** in `internal/tools/`; filename equals the tool name
  (`greet.go`, later `extract_csv.go`, etc.). Each file owns its `Input` /
  `Output` types and handler.
- **Registration seam:** `tools.Register(server)` adds every tool. `main.go`
  never calls `mcp.AddTool` directly, and stays untouched when tools are added.
- **`main.go` is lifecycle only** — no business logic.
- **Server identity:** `Name: "consensus"`.

## Verification

- `go build ./...` succeeds inside `mcp/`.
- The server speaks MCP over stdio: a scripted `initialize` → `tools/list` →
  `tools/call greet` handshake returns the expected greeting.

## Out of scope (this session)

- CSV / database ingestion, Postgres / SQLite integration, statistical tools.
  The layout leaves room for `internal/data/` and `internal/stats/`, but those
  directories are not created until their specs exist.
- Any code under `app/`.

## Follow-up

- Run `/init` against the real scaffold to generate `CLAUDE.md`.
