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
