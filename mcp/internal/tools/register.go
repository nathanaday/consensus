package tools

import "github.com/modelcontextprotocol/go-sdk/mcp"

// Register adds every Consensus tool to the server. New tools are added here
// and in their own file; main.go does not change.
func Register(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: "greet", Description: "say hi"}, SayHi)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ingest_csv",
		Description: "Ingest a local time-series CSV into the dataset store. Auto-detects the timestamp column and, from the first data row's numeric cells, the value columns (override with timestamp_col / value_cols; pass value_cols explicitly if a column's first value may be blank or non-numeric). Normalizes to a canonical long layout, stores it as Parquet, and returns a dataset_id plus a schema summary. row_count is the number of stored long-format rows (one per series per timestamp), not the number of source CSV timestamps. Reuse the dataset_id in later tools; it never returns row data. Pass units as a map of series id to unit of measurement to record them (e.g. {\"temp_c\":\"°C\"}); series without an entry are stored with no unit.",
	}, IngestCSV)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_datasets",
		Description: "List every dataset in the store with its metadata: id, kind, series (each with an optional unit of measurement), row_count (number of stored long-format rows), time_range, on-disk size_bytes, source_path, and created_at. Takes no arguments and returns no row data. An empty store returns an empty datasets list. A series with no recorded unit omits the unit field — report it as not recorded rather than inventing one.",
	}, ListDatasets)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "describe_dataset",
		Description: "Describe one dataset by id: its kind, series (each with an optional unit), row_count, time_range, on-disk size_bytes, source_path, created_at, origin (how it was made), its parent (the dataset it was copied from, or null for a root loaded from a file), and its children (datasets copied from it). Returns no row data. Use this to answer what a dataset was copied from and what was derived from it.",
	}, DescribeDataset)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "server_info",
		Description: "Report where and how the server stores data and what it can do: store_dir (the on-disk store location), files (the files currently in it), storage_format (the format datasets are written in), supported_ingest_formats (the source formats that can be ingested today), and capabilities (a short summary of current features). Takes no arguments.",
	}, ServerInfo)
}
