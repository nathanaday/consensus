package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestServerSpeaksMCPOverStdio(t *testing.T) {
	ctx := context.Background()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	transport := &mcp.CommandTransport{Command: exec.Command("go", "run", ".")}

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("connect to server subprocess: %v", err)
	}
	defer session.Close()

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "greet",
		Arguments: map[string]any{"name": "Grace"},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool returned an error result: %+v", res)
	}

	data, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	if !strings.Contains(string(data), "Hi Grace") {
		t.Fatalf("expected result to contain %q, got %s", "Hi Grace", data)
	}
}

func TestServerIngestsCSVOverStdio(t *testing.T) {
	ctx := context.Background()

	storeDir := t.TempDir()
	csvPath := filepath.Join(t.TempDir(), "readings.csv")
	csv := "time,temp_c\n2026-01-01T00:00:00Z,12.4\n2026-01-01T00:05:00Z,12.6\n"
	if err := os.WriteFile(csvPath, []byte(csv), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	cmd := exec.Command("go", "run", ".")
	cmd.Env = append(os.Environ(), "CONSENSUS_STORE_DIR="+storeDir)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatalf("connect to server subprocess: %v", err)
	}
	defer session.Close()

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "ingest_csv",
		Arguments: map[string]any{"path": csvPath},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool returned error result: %+v", res)
	}

	data, _ := json.Marshal(res)
	if !strings.Contains(string(data), `"dataset_id":"readings/temp_c"`) {
		t.Fatalf("expected dataset_id readings/temp_c, got %s", data)
	}
	if _, err := os.Stat(filepath.Join(storeDir, "readings", "temp_c.parquet")); err != nil {
		t.Fatalf("parquet not stored: %v", err)
	}
}

func TestServerIntrospectionOverStdio(t *testing.T) {
	ctx := context.Background()

	storeDir := t.TempDir()
	csvPath := filepath.Join(t.TempDir(), "readings.csv")
	csv := "time,temp_c\n2026-01-01T00:00:00Z,12.4\n2026-01-01T00:05:00Z,12.6\n"
	if err := os.WriteFile(csvPath, []byte(csv), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	cmd := exec.Command("go", "run", ".")
	cmd.Env = append(os.Environ(), "CONSENSUS_STORE_DIR="+storeDir)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatalf("connect to server subprocess: %v", err)
	}
	defer session.Close()

	if _, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "ingest_csv",
		Arguments: map[string]any{"path": csvPath},
	}); err != nil {
		t.Fatalf("ingest_csv: %v", err)
	}

	listRes, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "list_datasets"})
	if err != nil {
		t.Fatalf("list_datasets: %v", err)
	}
	if listRes.IsError {
		t.Fatalf("list_datasets error result: %+v", listRes)
	}
	listData, _ := json.Marshal(listRes)
	if !strings.Contains(string(listData), `"id":"readings/temp_c"`) {
		t.Fatalf("expected dataset readings/temp_c, got %s", listData)
	}

	infoRes, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "server_info"})
	if err != nil {
		t.Fatalf("server_info: %v", err)
	}
	if infoRes.IsError {
		t.Fatalf("server_info error result: %+v", infoRes)
	}
	infoData, _ := json.Marshal(infoRes)
	if !strings.Contains(string(infoData), `"storage_format":"parquet"`) {
		t.Fatalf("expected storage_format parquet, got %s", infoData)
	}
}

func TestServerLineageOverStdio(t *testing.T) {
	ctx := context.Background()

	storeDir := t.TempDir()
	csvPath := filepath.Join(t.TempDir(), "readings.csv")
	csv := "time,temp_c\n2026-01-01T00:00:00Z,12.4\n2026-01-01T00:05:00Z,12.6\n"
	if err := os.WriteFile(csvPath, []byte(csv), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	cmd := exec.Command("go", "run", ".")
	cmd.Env = append(os.Environ(), "CONSENSUS_STORE_DIR="+storeDir)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatalf("connect to server subprocess: %v", err)
	}
	defer session.Close()

	if _, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "ingest_csv",
		Arguments: map[string]any{"path": csvPath},
	}); err != nil {
		t.Fatalf("ingest_csv: %v", err)
	}

	copyRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "copy_dataset",
		Arguments: map[string]any{"id": "readings/temp_c"},
	})
	if err != nil {
		t.Fatalf("copy_dataset: %v", err)
	}
	if copyRes.IsError {
		t.Fatalf("copy_dataset error result: %+v", copyRes)
	}
	if !strings.Contains(string(mustMarshal(copyRes)), `"id":"readings/temp_c-2"`) {
		t.Fatalf("expected copy id readings/temp_c-2, got %s", mustMarshal(copyRes))
	}

	descRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "describe_dataset",
		Arguments: map[string]any{"id": "readings/temp_c-2"},
	})
	if err != nil {
		t.Fatalf("describe_dataset: %v", err)
	}
	if !strings.Contains(string(mustMarshal(descRes)), `"parent":{"id":"readings/temp_c","origin":"csv"}`) {
		t.Fatalf("expected parent edge, got %s", mustMarshal(descRes))
	}

	graphRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "dataset_graph",
		Arguments: map[string]any{"format": "mermaid"},
	})
	if err != nil {
		t.Fatalf("dataset_graph: %v", err)
	}
	// Assert on the copy edge's label + target, which contain no JSON-escaped
	// characters (the "-->" arrow's ">" is HTML-escaped on the wire, so match
	// the unescaped part of the edge instead).
	if !strings.Contains(string(mustMarshal(graphRes)), `|copy| readings_temp_c_2`) {
		t.Fatalf("expected copy edge in mermaid, got %s", mustMarshal(graphRes))
	}
}

func TestServerIngestsEpochCSVPerChannelOverStdio(t *testing.T) {
	ctx := context.Background()

	fixture, err := filepath.Abs(filepath.Join("..", "..", "..", "data", "iot_telemetry_data-reduced.csv"))
	if err != nil {
		t.Fatalf("resolve fixture path: %v", err)
	}
	if _, err := os.Stat(fixture); err != nil {
		t.Fatalf("fixture missing: %v", err)
	}

	storeDir := t.TempDir()
	cmd := exec.Command("go", "run", ".")
	cmd.Env = append(os.Environ(), "CONSENSUS_STORE_DIR="+storeDir)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatalf("connect to server subprocess: %v", err)
	}
	defer session.Close()

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "ingest_csv",
		Arguments: map[string]any{"path": fixture, "name": "iot"},
	})
	if err != nil {
		t.Fatalf("ingest_csv: %v", err)
	}
	if res.IsError {
		t.Fatalf("ingest_csv error result: %s", mustMarshal(res))
	}

	s := string(mustMarshal(res))
	for _, want := range []string{
		`"group":"iot"`,
		`"timestamp_column":"ts"`,
		`"dataset_id":"iot/humidity"`,
		`"dataset_id":"iot/smoke"`,
		`"dataset_id":"iot/temp"`,
		`"row_count":10000`,
		`"start":"2020-07-12T00:01:34Z"`,
		`"end":"2020-07-13T23:43:29Z"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("expected %s in ingest result: %s", want, s)
		}
	}

	for _, ch := range []string{"humidity", "smoke", "temp"} {
		if _, err := os.Stat(filepath.Join(storeDir, "iot", ch+".parquet")); err != nil {
			t.Errorf("channel parquet %s not stored: %v", ch, err)
		}
	}
}

func TestServerAnalysisOverStdio(t *testing.T) {
	ctx := context.Background()

	storeDir := t.TempDir()
	csvPath := filepath.Join(t.TempDir(), "readings.csv")
	csv := "ts,temp\n" +
		"2026-01-01T00:00:00Z,20.0\n" +
		"2026-01-01T00:01:00Z,21.0\n" +
		"2026-01-01T00:02:00Z,20.5\n" +
		"2026-01-01T00:03:00Z,90.0\n" +
		"2026-01-01T00:04:00Z,21.5\n"
	if err := os.WriteFile(csvPath, []byte(csv), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	// values {20,20.5,21,21.5,90}: Q1=20.5, Q3=21.5, IQR=1 -> bounds [19, 23]

	cmd := exec.Command("go", "run", ".")
	cmd.Env = append(os.Environ(), "CONSENSUS_STORE_DIR="+storeDir)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatalf("connect to server subprocess: %v", err)
	}
	defer session.Close()

	if _, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "ingest_csv",
		Arguments: map[string]any{"path": csvPath},
	}); err != nil {
		t.Fatalf("ingest_csv: %v", err)
	}

	sumRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "summary_stats", Arguments: map[string]any{"id": "readings/temp"},
	})
	if err != nil {
		t.Fatalf("summary_stats: %v", err)
	}
	if sumRes.IsError {
		t.Fatalf("summary_stats error result: %+v", sumRes)
	}
	if s := string(mustMarshal(sumRes)); !strings.Contains(s, `"row_count":5`) || !strings.Contains(s, `"value":90`) {
		t.Errorf("dirty summary missing expected stats: %s", s)
	}

	outRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "detect_outliers", Arguments: map[string]any{"id": "readings/temp"},
	})
	if err != nil {
		t.Fatalf("detect_outliers: %v", err)
	}
	if outRes.IsError {
		t.Fatalf("detect_outliers error result: %+v", outRes)
	}
	if s := string(mustMarshal(outRes)); !strings.Contains(s, `"total_outliers":1`) {
		t.Errorf("expected exactly one outlier: %s", s)
	}

	rmRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "remove_outliers", Arguments: map[string]any{"id": "readings/temp"},
	})
	if err != nil {
		t.Fatalf("remove_outliers: %v", err)
	}
	if rmRes.IsError {
		t.Fatalf("remove_outliers error result: %+v", rmRes)
	}
	s := string(mustMarshal(rmRes))
	if !strings.Contains(s, `"rows_removed":1`) {
		t.Errorf("expected one row removed: %s", s)
	}
	if !strings.Contains(s, `"id":"readings/temp-2"`) {
		t.Errorf("expected child id readings/temp-2: %s", s)
	}

	cleanRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "summary_stats", Arguments: map[string]any{"id": "readings/temp-2"},
	})
	if err != nil {
		t.Fatalf("summary_stats on child: %v", err)
	}
	if cleanRes.IsError {
		t.Fatalf("summary_stats on child error result: %+v", cleanRes)
	}
	if s := string(mustMarshal(cleanRes)); !strings.Contains(s, `"row_count":4`) || !strings.Contains(s, `"value":21.5`) {
		t.Errorf("cleaned summary should have 4 rows and max 21.5: %s", s)
	}
}

func mustMarshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
