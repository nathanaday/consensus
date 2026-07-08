package tools

import "github.com/modelcontextprotocol/go-sdk/mcp"

// Register adds every Consensus tool to the server. New tools are added here
// and in their own file; main.go does not change.
func Register(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: "greet", Description: "say hi"}, SayHi)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ingest_csv",
		Description: "Ingest a local time-series CSV into the dataset store, creating one dataset per numeric value column (dataset ids look like group/column, e.g. readings/temp_c). Auto-detects the timestamp column — string dates (RFC3339 and common layouts) or Unix epoch seconds/milliseconds/microseconds/nanoseconds, integer or float — preferring time-like column names; override with timestamp_col. Value columns are detected from the first data row's numeric cells (override with value_cols; pass it explicitly if a column's first value may be blank or non-numeric). Each channel is stored as Parquet with its own row_count and time_range (blank cells are skipped per channel). Returns the shared group id plus a summary per created dataset; it never returns row data. Pass units as a map of column name to unit of measurement (e.g. {\"temp_c\":\"°C\"}); columns without an entry are stored with no unit.",
	}, IngestCSV)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_datasets",
		Description: "List every dataset in the store with its metadata: id, kind, source_column (the source column the channel came from), unit (optional unit of measurement), row_count, time_range, on-disk size_bytes, source_path, and created_at. Each dataset is a single channel; datasets ingested from the same file share an id prefix (group/column) and source_path. Takes no arguments and returns no row data. An empty store returns an empty datasets list. A dataset with no recorded unit omits the unit field — report it as not recorded rather than inventing one.",
	}, ListDatasets)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "describe_dataset",
		Description: "Describe one dataset by id: its kind, source_column, unit (omitted when not recorded), row_count, time_range, on-disk size_bytes, source_path, created_at, origin (how it was made), its parent (the dataset it was copied from, or null for a root loaded from a file), and its children (datasets copied from it). Returns no row data. Use this to answer what a dataset was copied from and what was derived from it.",
	}, DescribeDataset)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "preview_dataset",
		Description: "Return a bounded sample of a dataset's rows (timestamp, value) so you can eyeball the data. limit defaults to 20 and is capped at 200 — this is a preview, not an export. Also reports returned (rows in this response) and row_count (total rows in the dataset).",
	}, PreviewDataset)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "copy_dataset",
		Description: "Create a new immutable copy of a dataset. The copy becomes a child of the source in the lineage graph (origin \"copy\"). Pass name to choose the new id; otherwise it is derived from the source id and disambiguated (e.g. readings -> readings-2). Returns the new dataset's description, including its parent edge.",
	}, CopyDataset)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "dataset_graph",
		Description: "Render the lineage of all datasets. format \"mermaid\" (default) returns a Mermaid flowchart string (graph TD) with an edge per copy labeled by origin; format \"json\" returns a structured adjacency graph under \"graph\" with roots and a node map (parent_id, origin, children). Takes no other arguments; an empty store returns an empty graph.",
	}, DatasetGraph)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "server_info",
		Description: "Report where and how the server stores data and what it can do: store_dir (the on-disk store location), files (the files currently in it), storage_format (the format datasets are written in), supported_ingest_formats (the source formats that can be ingested today), and capabilities (a short summary of current features). Takes no arguments.",
	}, ServerInfo)
}
