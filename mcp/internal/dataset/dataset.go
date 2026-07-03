// Package dataset holds the value types shared between ingestion and storage.
package dataset

// Row is one measurement in the canonical long layout. Timestamp is UTC epoch
// milliseconds.
type Row struct {
	Timestamp int64   `parquet:"timestamp" json:"timestamp"`
	SeriesID  string  `parquet:"series_id" json:"series_id"`
	Value     float64 `parquet:"value" json:"value"`
}

// TimeRange is the inclusive span of a dataset, RFC3339 UTC strings.
type TimeRange struct {
	Start string `json:"start"`
	End   string `json:"end"`
}
