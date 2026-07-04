package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/store"
)

type ServerInfoInput struct{}

type ServerInfoOutput struct {
	StoreDir               string   `json:"store_dir"`
	Files                  []string `json:"files"`
	StorageFormat          string   `json:"storage_format"`
	SupportedIngestFormats []string `json:"supported_ingest_formats"`
	Capabilities           []string `json:"capabilities"`
}

// ServerInfo reports where and how the server stores data and what it can do.
// The store location and file list are read live; the formats and capabilities
// reflect what the code actually supports today.
func ServerInfo(ctx context.Context, req *mcp.CallToolRequest, input ServerInfoInput) (*mcp.CallToolResult, ServerInfoOutput, error) {
	cfg, err := store.Resolve()
	if err != nil {
		return nil, ServerInfoOutput{}, err
	}
	files, err := store.ListStoreFiles(cfg.Dir)
	if err != nil {
		return nil, ServerInfoOutput{}, err
	}

	return nil, ServerInfoOutput{
		StoreDir:               cfg.Dir,
		Files:                  files,
		StorageFormat:          store.StorageFormat,
		SupportedIngestFormats: store.SupportedIngestFormats(),
		Capabilities:           store.Capabilities(),
	}, nil
}
