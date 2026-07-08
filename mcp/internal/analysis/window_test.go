package analysis

import (
	"strings"
	"testing"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

func threeRows() []dataset.Row {
	return []dataset.Row{
		{Timestamp: 0, Value: 1},
		{Timestamp: 60000, Value: 2},
		{Timestamp: 120000, Value: 3},
	}
}

func TestWindowBoundsAreInclusive(t *testing.T) {
	got, err := Window(threeRows(), "1970-01-01T00:01:00Z", "1970-01-01T00:02:00Z")
	if err != nil {
		t.Fatalf("Window: %v", err)
	}
	if len(got) != 2 || got[0].Timestamp != 60000 || got[1].Timestamp != 120000 {
		t.Errorf("windowed rows = %+v, want timestamps 60000 and 120000", got)
	}
}

func TestWindowEmptySidesAreUnbounded(t *testing.T) {
	got, err := Window(threeRows(), "", "")
	if err != nil {
		t.Fatalf("Window: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("unbounded window kept %d rows, want 3", len(got))
	}
	got, err = Window(threeRows(), "1970-01-01T00:01:30Z", "")
	if err != nil {
		t.Fatalf("Window: %v", err)
	}
	if len(got) != 1 || got[0].Timestamp != 120000 {
		t.Errorf("start-only window = %+v, want just timestamp 120000", got)
	}
}

func TestWindowSortsOutOfOrderInput(t *testing.T) {
	rows := []dataset.Row{{Timestamp: 120000, Value: 3}, {Timestamp: 0, Value: 1}}
	got, err := Window(rows, "", "")
	if err != nil {
		t.Fatalf("Window: %v", err)
	}
	if got[0].Timestamp != 0 {
		t.Errorf("window did not sort: %+v", got)
	}
}

func TestWindowRejectsBadInputs(t *testing.T) {
	if _, err := Window(threeRows(), "yesterday", ""); err == nil || !strings.Contains(err.Error(), "yesterday") {
		t.Errorf("bad start error = %v, want it to name the value", err)
	}
	if _, err := Window(threeRows(), "", "not-a-time"); err == nil || !strings.Contains(err.Error(), "not-a-time") {
		t.Errorf("bad end error = %v, want it to name the value", err)
	}
	if _, err := Window(threeRows(), "1970-01-02T00:00:00Z", "1970-01-01T00:00:00Z"); err == nil || !strings.Contains(err.Error(), "after") {
		t.Errorf("start-after-end error = %v, want 'after'", err)
	}
}
