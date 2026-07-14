package tools

import (
	"fmt"
	"time"
)

// parseBucketMS parses a Go duration string (e.g. "1h", "15m") into whole
// milliseconds. The duration must be positive.
func parseBucketMS(s string) (int64, error) {
	if s == "" {
		return 0, fmt.Errorf("bucket is required; pass a Go duration like \"1h\", \"15m\", or \"30s\"")
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid bucket %q; expected a Go duration like \"1h\", \"15m\", or \"30s\"", s)
	}
	if d <= 0 {
		return 0, fmt.Errorf("bucket must be positive, got %q", s)
	}
	return d.Milliseconds(), nil
}
