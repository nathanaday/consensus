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
