package tools

import (
	"context"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/analysis"
	"github.com/nathanaday/consensus/mcp/internal/dataset"
	"github.com/nathanaday/consensus/mcp/internal/lineage"
)

type OverviewInput struct {
	Prefix string `json:"prefix,omitempty" jsonschema:"optional id prefix filter, e.g. a group id like readings; omitted covers every dataset"`
}

type ChannelOverview struct {
	ID            string            `json:"id"`
	Unit          string            `json:"unit,omitempty"`
	Origin        string            `json:"origin,omitempty"`
	RowCount      int               `json:"row_count"`
	TimeRange     dataset.TimeRange `json:"time_range"`
	LastValue     *float64          `json:"last_value,omitempty"`
	LastTimestamp string            `json:"last_timestamp,omitempty"`
	Mean          *float64          `json:"mean,omitempty"`
	Min           *float64          `json:"min,omitempty"`
	Max           *float64          `json:"max,omitempty"`
	Stddev        *float64          `json:"stddev,omitempty"`
}

type OverviewOutput struct {
	DatasetCount int               `json:"dataset_count"`
	Channels     []ChannelOverview `json:"channels"`
}

// Overview reports a compact statistical snapshot of every channel in one
// call: latest reading plus headline stats per dataset. It returns
// statistics only, never row data.
func Overview(ctx context.Context, req *mcp.CallToolRequest, input OverviewInput) (*mcp.CallToolResult, OverviewOutput, error) {
	g, err := lineage.Open()
	if err != nil {
		return nil, OverviewOutput{}, err
	}

	channels := make([]ChannelOverview, 0)
	for _, n := range g.Nodes() {
		if input.Prefix != "" && !strings.HasPrefix(n.ID(), input.Prefix) {
			continue
		}
		e := n.Info()
		ch := ChannelOverview{
			ID:        n.ID(),
			Unit:      e.Unit,
			Origin:    e.Origin,
			RowCount:  e.RowCount,
			TimeRange: e.TimeRange,
		}
		rows, err := n.LoadData()
		if err != nil {
			return nil, OverviewOutput{}, err
		}
		if len(rows) > 0 {
			stats, serr := analysis.Summary(rows)
			if serr != nil {
				return nil, OverviewOutput{}, serr
			}
			last := rows[0]
			for _, r := range rows {
				if r.Timestamp >= last.Timestamp {
					last = r
				}
			}
			mean, min, max, sd := stats.Mean, stats.Min.Value, stats.Max.Value, stats.Stddev
			lv := last.Value
			ch.LastValue = &lv
			ch.LastTimestamp = renderMS(last.Timestamp)
			ch.Mean, ch.Min, ch.Max, ch.Stddev = &mean, &min, &max, &sd
		}
		channels = append(channels, ch)
	}
	return nil, OverviewOutput{DatasetCount: len(channels), Channels: channels}, nil
}
