package tools

import "testing"

func TestParseBucketMS(t *testing.T) {
	cases := map[string]int64{
		"1h":  3600000,
		"15m": 900000,
		"30s": 30000,
	}
	for in, want := range cases {
		got, err := parseBucketMS(in)
		if err != nil {
			t.Errorf("%q: unexpected error %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("%q: want %d, got %d", in, want, got)
		}
	}
}

func TestParseBucketMSErrors(t *testing.T) {
	for _, in := range []string{"", "nonsense", "0s", "-5m"} {
		if _, err := parseBucketMS(in); err == nil {
			t.Errorf("%q should error", in)
		}
	}
}
