// Package dataset holds the value types shared between ingestion and storage.
package dataset

// Row is one measurement in the canonical long layout. Timestamp is UTC epoch
// milliseconds.
type Row struct {
	Timestamp int64   `parquet:"timestamp,timestamp(millisecond:utc)" json:"timestamp"`
	SeriesID  string  `parquet:"series_id" json:"series_id"`
	Value     float64 `parquet:"value" json:"value"`
}

// TimeRange is the inclusive span of a dataset, RFC3339 UTC strings.
type TimeRange struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// Entry is a catalog record describing one stored dataset. It records schema
// and stats only, never data values.
type Entry struct {
	ID              string   `json:"id"`
	Kind            string   `json:"kind"`
	SourcePath      string   `json:"source_path"`
	CreatedAt       string   `json:"created_at"`
	TimestampColumn string   `json:"timestamp_column"`
	SeriesIDs       []string `json:"series_ids"`
	// RowCount is the number of canonical long-format rows (one per series
	// per timestamp), not the number of source CSV timestamps.
	RowCount  int       `json:"row_count"`
	TimeRange TimeRange `json:"time_range"`
}
