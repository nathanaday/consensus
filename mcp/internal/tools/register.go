package tools

import "github.com/modelcontextprotocol/go-sdk/mcp"

// Register adds every Consensus tool to the server. New tools are added here
// and in their own file; main.go does not change.
func Register(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: "greet", Description: "say hi"}, SayHi)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ingest_csv",
		Description: "Ingest a local time-series CSV into the dataset store. Auto-detects the timestamp column and numeric value columns (override with timestamp_col / value_cols). Normalizes to a canonical long layout, stores it as Parquet, and returns a dataset_id plus a schema summary. Reuse the dataset_id in later tools; it never returns row data.",
	}, IngestCSV)
}
