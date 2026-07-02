# Consensus MCP Server Scaffold Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stand up a hello-world MCP server for Consensus using the Go MCP SDK, laid out so Phase 1 time-series tools drop in without restructuring.

**Architecture:** Monorepo with two surfaces — `mcp/` (all Go) and `app/` (future). The Go module keeps `cmd/server/main.go` for process lifecycle only and puts every tool in `internal/tools/`, one file per tool. A single `tools.Register(server)` seam adds all tools so `main.go` never changes as tools are added. Correctness is proven two ways: an in-memory client/server test and a real subprocess stdio test.

**Tech Stack:** Go 1.25 (required by the MCP SDK; obtained via `GOTOOLCHAIN=auto` if the local default is older), `github.com/modelcontextprotocol/go-sdk/mcp` (v1.6.1).

## Global Constraints

- Go module path: `github.com/nathanaday/consensus/mcp` (go.mod lives in `mcp/`).
- MCP server identity: `Name: "consensus"`.
- One tool per file in `internal/tools/`; filename equals the tool name. Each tool file owns its `Input`/`Output` types and handler.
- `main.go` is lifecycle only — no business logic, no direct `mcp.AddTool` calls.
- All tools are registered through `tools.Register(server *mcp.Server)`.
- READMEs are user-owned; do not rewrite `mcp/README.md`.
- Commit after each task.

---

### Task 1: Repository restructure and Go module setup

Establishes the `mcp/` + `app/` layout, moves the existing README, initializes the Go module, pulls the SDK, and records project conventions. Deliverable: an initialized module with dependencies and docs in place.

**Files:**
- Move: `README.md` -> `mcp/README.md`
- Create: `app/README.md`
- Create: `mcp/CONVENTIONS.md`
- Create: `mcp/go.mod` (via `go mod init`)
- Create: `mcp/go.sum` (via `go get`)

**Interfaces:**
- Consumes: nothing (first task).
- Produces: module `github.com/nathanaday/consensus/mcp` with the SDK available for import at `github.com/modelcontextprotocol/go-sdk/mcp`.

- [ ] **Step 1: Move the existing README into `mcp/`**

```bash
cd /Users/nathanaday/SoftwareProjects/consensus
mkdir -p mcp app
git mv README.md mcp/README.md
```

- [ ] **Step 2: Create the `app/` placeholder README**

Create `app/README.md` with exactly:

```markdown
# Consensus App

Placeholder for the future full-stack Consensus application. Not yet designed.
```

- [ ] **Step 3: Initialize the Go module**

```bash
cd /Users/nathanaday/SoftwareProjects/consensus/mcp
go mod init github.com/nathanaday/consensus/mcp
```

Expected: creates `mcp/go.mod` containing `module github.com/nathanaday/consensus/mcp`.

- [ ] **Step 4: Add the MCP SDK dependency**

```bash
cd /Users/nathanaday/SoftwareProjects/consensus/mcp
go get github.com/modelcontextprotocol/go-sdk/mcp@latest
```

Expected: `go.mod` gains a `require github.com/modelcontextprotocol/go-sdk v1.6.1` (or newer) line and `go.sum` is created.

- [ ] **Step 5: Write the conventions doc**

Create `mcp/CONVENTIONS.md` with exactly:

```markdown
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
```

- [ ] **Step 6: Verify the module is initialized**

```bash
cd /Users/nathanaday/SoftwareProjects/consensus/mcp
head -n 1 go.mod
```

Expected: `module github.com/nathanaday/consensus/mcp`

- [ ] **Step 7: Commit**

```bash
cd /Users/nathanaday/SoftwareProjects/consensus
git add -A
git commit -m "Scaffold mcp/ + app/ layout and init Go module"
```

---

### Task 2: Greet tool, registration seam, and entry point

Implements the placeholder `greet` tool test-first using the SDK's in-memory transport, then the tool, the registration seam, and the lifecycle-only entry point. Deliverable: the module builds and the tool is reachable through a real client session.

**Files:**
- Test: `mcp/internal/tools/greet_test.go`
- Create: `mcp/internal/tools/greet.go`
- Create: `mcp/internal/tools/register.go`
- Create: `mcp/cmd/server/main.go`

**Interfaces:**
- Consumes: module and SDK from Task 1.
- Produces:
  - `tools.SayHi(ctx context.Context, req *mcp.CallToolRequest, input GreetInput) (*mcp.CallToolResult, GreetOutput, error)`
  - `tools.GreetInput{ Name string }`, `tools.GreetOutput{ Greeting string }`
  - `tools.Register(server *mcp.Server)` — adds the `greet` tool.

- [ ] **Step 1: Write the failing test**

Create `mcp/internal/tools/greet_test.go`:

```go
package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestGreetToolReturnsGreeting(t *testing.T) {
	ctx := context.Background()

	server := mcp.NewServer(&mcp.Implementation{Name: "consensus", Version: "test"}, nil)
	Register(server)

	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	if _, err := server.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatalf("server connect: %v", err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer session.Close()

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "greet",
		Arguments: map[string]any{"name": "Ada"},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool returned an error result: %+v", res)
	}

	data, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	if !strings.Contains(string(data), "Hi Ada") {
		t.Fatalf("expected result to contain %q, got %s", "Hi Ada", data)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd /Users/nathanaday/SoftwareProjects/consensus/mcp
go test ./internal/tools/ -run TestGreetToolReturnsGreeting -v
```

Expected: FAIL — build error, `undefined: Register` (and the package has no non-test files yet).

- [ ] **Step 3: Implement the greet tool**

Create `mcp/internal/tools/greet.go`:

```go
package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GreetInput struct {
	Name string `json:"name" jsonschema:"the name of the person to greet"`
}

type GreetOutput struct {
	Greeting string `json:"greeting" jsonschema:"the greeting to tell to the user"`
}

func SayHi(ctx context.Context, req *mcp.CallToolRequest, input GreetInput) (*mcp.CallToolResult, GreetOutput, error) {
	return nil, GreetOutput{Greeting: "Hi " + input.Name}, nil
}
```

- [ ] **Step 4: Implement the registration seam**

Create `mcp/internal/tools/register.go`:

```go
package tools

import "github.com/modelcontextprotocol/go-sdk/mcp"

// Register adds every Consensus tool to the server. New tools are added here
// and in their own file; main.go does not change.
func Register(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: "greet", Description: "say hi"}, SayHi)
}
```

- [ ] **Step 5: Implement the entry point**

Create `mcp/cmd/server/main.go`:

```go
package main

import (
	"context"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/nathanaday/consensus/mcp/internal/tools"
)

func main() {
	server := mcp.NewServer(&mcp.Implementation{Name: "consensus", Version: "v0.1.0"}, nil)
	tools.Register(server)
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
```

- [ ] **Step 6: Build the whole module**

```bash
cd /Users/nathanaday/SoftwareProjects/consensus/mcp
go build ./...
```

Expected: no output, exit code 0.

- [ ] **Step 7: Run the test to verify it passes**

```bash
cd /Users/nathanaday/SoftwareProjects/consensus/mcp
go test ./internal/tools/ -run TestGreetToolReturnsGreeting -v
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
cd /Users/nathanaday/SoftwareProjects/consensus
git add -A
git commit -m "Add greet tool, registration seam, and server entry point"
```

---

### Task 3: End-to-end stdio verification

Proves the built server actually speaks MCP over stdio (not just in-memory) by spawning it as a subprocess and calling the tool through a `CommandTransport`. Deliverable: a passing integration test exercising the real binary.

**Files:**
- Test: `mcp/cmd/server/integration_test.go`

**Interfaces:**
- Consumes: the `main` package server from Task 2 (run via `go run .`).
- Produces: nothing consumed by later tasks.

- [ ] **Step 1: Write the failing integration test**

Create `mcp/cmd/server/integration_test.go`:

```go
package main

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestServerSpeaksMCPOverStdio(t *testing.T) {
	ctx := context.Background()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	transport := &mcp.CommandTransport{Command: exec.Command("go", "run", ".")}

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("connect to server subprocess: %v", err)
	}
	defer session.Close()

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "greet",
		Arguments: map[string]any{"name": "Grace"},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool returned an error result: %+v", res)
	}

	data, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	if !strings.Contains(string(data), "Hi Grace") {
		t.Fatalf("expected result to contain %q, got %s", "Hi Grace", data)
	}
}
```

- [ ] **Step 2: Run the test to verify it passes**

```bash
cd /Users/nathanaday/SoftwareProjects/consensus/mcp
go test ./cmd/server/ -run TestServerSpeaksMCPOverStdio -v
```

Expected: PASS (the subprocess starts via `go run .`, completes the MCP handshake, and returns the greeting). Note: first run may be slower while `go run` compiles.

- [ ] **Step 3: Run the full test suite**

```bash
cd /Users/nathanaday/SoftwareProjects/consensus/mcp
go test ./...
```

Expected: all packages PASS.

- [ ] **Step 4: Commit**

```bash
cd /Users/nathanaday/SoftwareProjects/consensus
git add -A
git commit -m "Add end-to-end stdio integration test for the MCP server"
```

---

## Follow-up (post-plan, user-driven)

- Run `/init` against the completed scaffold to generate `CLAUDE.md`.

## Self-Review

- **Spec coverage:** Monorepo `mcp/`+`app/` layout (Task 1) ✓; README moved (Task 1 Step 1) ✓; module path `github.com/nathanaday/consensus/mcp` (Task 1 Step 3) ✓; `CONVENTIONS.md` with all four preferences (Task 1 Step 5) ✓; greet tool + one-file-per-tool (Task 2) ✓; `tools.Register` seam, `main.go` lifecycle-only (Task 2) ✓; server `Name: "consensus"` (Task 2 Step 5) ✓; `go build ./...` gate (Task 2 Step 6) ✓; stdio `initialize`→tool-call handshake (Task 3, via CommandTransport) ✓; `app/` stub, no code (Task 1 Step 2) ✓; `/init` follow-up noted ✓. No gaps.
- **Placeholder scan:** No TBD/TODO/"add error handling"; every code and command step shows real content.
- **Type consistency:** `GreetInput`/`GreetOutput`/`SayHi`/`Register` names and signatures match across `greet.go`, `register.go`, `main.go`, and both tests. Tool name `"greet"` and argument key `"name"` are consistent everywhere.
