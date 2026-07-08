// Package dataset holds the value types shared between ingestion and storage.
package dataset

// Row is one measurement in a single-channel dataset. Timestamp is UTC epoch
// milliseconds.
type Row struct {
	Timestamp int64   `parquet:"timestamp,timestamp(millisecond:utc)" json:"timestamp"`
	Value     float64 `parquet:"value" json:"value"`
}

// TimeRange is the inclusive span of a dataset, RFC3339 UTC strings.
type TimeRange struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// Entry is a catalog record describing one stored dataset — a single channel
// of measurements. It records schema and stats only, never data values.
type Entry struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	SourcePath string `json:"source_path"`
	// SourceColumn is the source column this channel came from.
	SourceColumn string `json:"source_column"`
	// Unit is the channel's unit of measurement; empty when not recorded.
	Unit            string `json:"unit,omitempty"`
	CreatedAt       string `json:"created_at"`
	TimestampColumn string `json:"timestamp_column"`
	// RowCount is the number of stored rows in this channel; blank source
	// cells are skipped, so channels of one ingest can differ.
	RowCount  int       `json:"row_count"`
	TimeRange TimeRange `json:"time_range"`
	// ParentID is the id of the dataset this one was copied/derived from;
	// "" marks a root loaded from a source file.
	ParentID string `json:"parent_id"`
	// Origin describes how this dataset came to be: "csv" for a root ingest,
	// "copy" for a plain copy, or a transform description for a derived dataset.
	Origin string `json:"origin,omitempty"`
}
