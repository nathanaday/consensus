# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository shape

Consensus is a monorepo with two surfaces:

- `mcp/` — the Go MCP server (all Go code lives here). This is the only active surface today.
- `app/` — placeholder for a future full-stack app; not yet designed, contains only a stub README.

The project is an MCP server for natural-language interaction with time-series / IoT data. The current code is a hello-world scaffold; Phase 1 (CSV / Postgres / SQLite ingestion and time-series statistical tools) is not yet built. See `mcp/README.md` for the product concept and `docs/superpowers/` for design specs and implementation plans.

## Commands

All Go commands run from the `mcp/` directory (that is where `go.mod` lives).

```bash
cd mcp
go build ./...                                              # build
go test ./...                                              # all tests
go test ./internal/tools/ -run TestGreetToolReturnsGreeting -v   # single test
go run ./cmd/server                                        # run the MCP server (serves over stdio)
go mod tidy                                                # after adding/removing imports
```

The server speaks MCP over stdin/stdout; it is meant to be launched by an MCP client, not run interactively.

**Go 1.25 is required** — the MCP SDK (`github.com/modelcontextprotocol/go-sdk`) mandates it. With the default `GOTOOLCHAIN=auto`, Go fetches the right toolchain automatically even if the local default is older.

## Architecture

The server is built on the Go MCP SDK. Two rules govern how tools are wired in — they are enforced conventions, documented in `mcp/CONVENTIONS.md`:

- **One tool per file** in `internal/tools/`, filename equal to the tool name. Each file owns its `Input`/`Output` types and its handler.
- **Single registration seam:** `tools.Register(server)` is the only place `mcp.AddTool` is called. `cmd/server/main.go` is lifecycle-only (construct server with `Name: "consensus"`, call `tools.Register`, run the transport) and must not change when new tools are added.

When adding a tool: create `internal/tools/<name>.go` with its types and handler, then add one `mcp.AddTool` line to `internal/tools/register.go`. Nothing else needs editing.

Tests use the SDK two ways: an in-memory transport for fast per-tool round-trips (`internal/tools/*_test.go`) and a real subprocess over stdio via `CommandTransport` running `go run .` for end-to-end verification (`cmd/server/integration_test.go`).

## Workflow notes

- READMEs are user-owned — do not create or rewrite `README.md` files unless asked.
- Design specs live in `docs/superpowers/specs/`, implementation plans in `docs/superpowers/plans/`.
