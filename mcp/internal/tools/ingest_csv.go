package tools

import (
	"context"
	"fmt"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/ingest"
	"github.com/nathanaday/consensus/mcp/internal/store"
)

type IngestCSVInput struct {
	Path         string   `json:"path" jsonschema:"local filesystem path to the CSV file to ingest"`
	Name         string   `json:"name,omitempty" jsonschema:"optional dataset id; defaults to a slug of the filename"`
	TimestampCol string   `json:"timestamp_col,omitempty" jsonschema:"column to use as the timestamp; auto-detected when omitted"`
	ValueCols    []string `json:"value_cols,omitempty" jsonschema:"columns to treat as value series; auto-detected (all numeric columns) when omitted"`
}

type IngestCSVTimeRange struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type IngestCSVDetected struct {
	TimestampColumn string   `json:"timestamp_column"`
	ValueColumns    []string `json:"value_columns"`
}

type IngestCSVOutput struct {
	DatasetID string             `json:"dataset_id"`
	SeriesIDs []string           `json:"series_ids"`
	RowCount  int                `json:"row_count"`
	TimeRange IngestCSVTimeRange `json:"time_range"`
	Detected  IngestCSVDetected  `json:"detected"`
}

// IngestCSV reads a time-series CSV, normalizes it to the canonical long layout,
// stores it as Parquet, records it in the catalog, and returns a schema summary.
func IngestCSV(ctx context.Context, req *mcp.CallToolRequest, input IngestCSVInput) (*mcp.CallToolResult, IngestCSVOutput, error) {
	f, err := os.Open(input.Path)
	if err != nil {
		return nil, IngestCSVOutput{}, fmt.Errorf("open csv %q: %w", input.Path, err)
	}
	defer f.Close()

	res, err := ingest.FromCSV(f, ingest.Options{TimestampCol: input.TimestampCol, ValueCols: input.ValueCols})
	if err != nil {
		return nil, IngestCSVOutput{}, err
	}

	cfg, err := store.Resolve()
	if err != nil {
		return nil, IngestCSVOutput{}, err
	}

	entry, err := store.SaveDataset(cfg, store.SaveRequest{
		NameOverride:    input.Name,
		SourcePath:      input.Path,
		TimestampColumn: res.TimestampColumn,
		SeriesIDs:       res.SeriesIDs,
		RowCount:        res.RowCount,
		TimeRange:       res.TimeRange,
		Rows:            res.Rows,
	})
	if err != nil {
		return nil, IngestCSVOutput{}, err
	}

	return nil, IngestCSVOutput{
		DatasetID: entry.ID,
		SeriesIDs: entry.SeriesIDs,
		RowCount:  entry.RowCount,
		TimeRange: IngestCSVTimeRange{Start: entry.TimeRange.Start, End: entry.TimeRange.End},
		Detected:  IngestCSVDetected{TimestampColumn: res.TimestampColumn, ValueColumns: res.ValueColumns},
	}, nil
}
