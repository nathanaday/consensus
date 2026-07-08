package tools

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/ingest"
	"github.com/nathanaday/consensus/mcp/internal/store"
)

type IngestCSVInput struct {
	Path         string            `json:"path" jsonschema:"local filesystem path to the CSV file to ingest"`
	Name         string            `json:"name,omitempty" jsonschema:"optional group id for the created datasets; defaults to a slug of the filename"`
	TimestampCol string            `json:"timestamp_col,omitempty" jsonschema:"column to use as the timestamp; auto-detected when omitted"`
	ValueCols    []string          `json:"value_cols,omitempty" jsonschema:"columns to ingest as channels; auto-detected when omitted from the first data row's numeric cells. Pass explicitly if a column's first value may be blank or non-numeric"`
	Units        map[string]string `json:"units,omitempty" jsonschema:"optional map of column name to unit of measurement (e.g. {\"temp_c\":\"°C\"}); columns without an entry are recorded with no unit"`
}

type IngestCSVTimeRange struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type IngestCSVDetected struct {
	TimestampColumn string   `json:"timestamp_column"`
	ValueColumns    []string `json:"value_columns"`
}

type IngestCSVDataset struct {
	DatasetID string             `json:"dataset_id"`
	Column    string             `json:"column"`
	Unit      string             `json:"unit,omitempty"`
	RowCount  int                `json:"row_count" jsonschema:"number of stored rows in this channel; blank source cells are skipped"`
	TimeRange IngestCSVTimeRange `json:"time_range"`
}

type IngestCSVOutput struct {
	Group    string             `json:"group" jsonschema:"the id prefix shared by every dataset created from this file"`
	Detected IngestCSVDetected  `json:"detected"`
	Datasets []IngestCSVDataset `json:"datasets"`
}

// IngestCSV reads a time-series CSV, splits it into one dataset per value
// column, stores each channel as Parquet, and returns a per-dataset summary.
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

	chans := make([]store.ChannelData, 0, len(res.Channels))
	valueColumns := make([]string, 0, len(res.Channels))
	for _, ch := range res.Channels {
		valueColumns = append(valueColumns, ch.Column)
		chans = append(chans, store.ChannelData{
			Column:    ch.Column,
			Unit:      input.Units[ch.Column],
			Rows:      ch.Rows,
			RowCount:  ch.RowCount,
			TimeRange: ch.TimeRange,
		})
	}

	entries, err := store.SaveGroup(cfg, store.GroupRequest{
		NameOverride:    input.Name,
		SourcePath:      input.Path,
		TimestampColumn: res.TimestampColumn,
		Channels:        chans,
		Origin:          "csv",
	})
	if err != nil {
		return nil, IngestCSVOutput{}, err
	}

	datasets := make([]IngestCSVDataset, 0, len(entries))
	for _, e := range entries {
		datasets = append(datasets, IngestCSVDataset{
			DatasetID: e.ID,
			Column:    e.SourceColumn,
			Unit:      e.Unit,
			RowCount:  e.RowCount,
			TimeRange: IngestCSVTimeRange{Start: e.TimeRange.Start, End: e.TimeRange.End},
		})
	}

	group := entries[0].ID
	if i := strings.Index(group, "/"); i >= 0 {
		group = group[:i]
	}

	return nil, IngestCSVOutput{
		Group:    group,
		Detected: IngestCSVDetected{TimestampColumn: res.TimestampColumn, ValueColumns: valueColumns},
		Datasets: datasets,
	}, nil
}
