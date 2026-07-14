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
	mcp.AddTool(server, &mcp.Tool{
		Name:        "summary_stats",
		Description: "Compute descriptive statistics for one dataset: row_count, min and max (each with the timestamp it occurred at), mean, median, population stddev, the unit if recorded, and analyzed_range (the actual span examined). Pass optional start/end (RFC3339 UTC) to analyze only that inclusive time window. Returns statistics only, never row data.",
	}, SummaryStats)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "rate_of_change",
		Description: "Compute rate-of-change statistics for one dataset from the pairwise derivative between consecutive points, in value units per second: max_rise and max_fall (each with the timestamp where the interval ends), mean_abs_rate, and median_sample_interval_seconds (use it to judge how evenly the data is sampled). Pairs sharing a timestamp are skipped. Pass optional start/end (RFC3339 UTC) to analyze only that inclusive time window. Needs at least 2 rows. Returns statistics only, never row data.",
	}, RateOfChange)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "detect_outliers",
		Description: "Find outliers in one dataset using the IQR method: a point outside [Q1 - k*IQR, Q3 + k*IQR] is an outlier (k = iqr_multiplier, default 1.5). Returns the quartiles and bounds used, total_outliers and percent of rows, plus the most extreme points (timestamp, value, deviation beyond the bound), capped at limit (default 20, max 100) — never bulk row data. Pass optional start/end (RFC3339 UTC) to analyze only that inclusive time window.",
	}, DetectOutliers)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "remove_outliers",
		Description: "Create a new dataset containing a source dataset's inliers: points outside the IQR fence [Q1 - k*IQR, Q3 + k*IQR] are dropped (k = iqr_multiplier, default 1.5). The new dataset is an immutable child of the source in the lineage graph (origin \"remove_outliers\"). Pass optional start/end (RFC3339 UTC) to first window the source, so this also works as a time-slice; bounds are computed over the window. Pass name to choose the new id, otherwise it is derived from the source id. Returns the new dataset's description plus rows_removed and the bounds applied — never row data. A run that removes nothing still creates the dataset.",
	}, RemoveOutliers)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "profile",
		Description: "Summarize the shape of one dataset over time as up to 48 time buckets, each with mean, min, max, and count. Pass bucket as a Go duration (e.g. 1h, 15m) or omit it to auto-pick a round width. Empty buckets appear with count 0 so gaps are visible. Pass optional start/end (RFC3339 UTC) to profile only that window. Returns bucketed statistics only, never row data.",
	}, Profile)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "resample",
		Description: "Create a new dataset by bucketing a source dataset into fixed-width intervals and aggregating each (agg: mean default, min, max, or median). bucket is a required Go duration (e.g. 1h) and must be at least the source's median sampling interval. The new dataset is an immutable child of the source in the lineage graph (origin \"resample\"); empty buckets are dropped. Pass optional start/end (RFC3339 UTC) to window the source first, and name to choose the new id. Use it to smooth noise or shrink a series before trend analysis. Returns the new dataset's description plus the bucket and agg used — never row data.",
	}, Resample)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "fit_trend",
		Description: "Fit a linear trend to one dataset over time by least squares. Returns direction (increasing, decreasing, or flat when the slope is not statistically distinguishable from zero), slope_per_hour and slope_per_day (in value units), r_squared (goodness of fit), window_duration_seconds, and caveats about short windows, few points, or weak fit. Pass optional start/end (RFC3339 UTC) to fit only that window. For noisy data, remove_outliers or resample first, then fit_trend on the result. Returns statistics only, never row data.",
	}, FitTrend)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "compare_to_baseline",
		Description: "Compare a subject window of one dataset against a baseline distribution and report anomalies. The baseline defaults to all history before the subject window (override with baseline_id, baseline_start, baseline_end). Points outside the baseline's IQR fence are grouped into episodes, each with its interval, direction, peak value and time, and deviation (top 10 by deviation; total_episodes is the full count). Also returns the baseline and subject summaries, points_outside, and pct_outside. An empty episode list means the subject is within the typical range. Pass start/end (RFC3339 UTC) for the subject window. Returns statistics only, never row data.",
	}, CompareToBaseline)
}
