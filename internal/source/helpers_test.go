package source

import (
	"testing"
	"time"
)

func filterByMetric(records []Record, metric string) []Record {
	var filtered []Record
	for _, r := range records {
		if r.Metric == metric {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func TestInDateRange(t *testing.T) {
	start := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		t    time.Time
		want bool
	}{
		{"before start", time.Date(2025, 1, 14, 23, 59, 59, 0, time.UTC), false},
		{"at start", time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC), true},
		{"mid range", time.Date(2025, 1, 17, 12, 0, 0, 0, time.UTC), true},
		{"end of last day", time.Date(2025, 1, 20, 23, 59, 59, 0, time.UTC), true},
		{"after end", time.Date(2025, 1, 21, 0, 0, 0, 0, time.UTC), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := inDateRange(tt.t, start, end); got != tt.want {
				t.Errorf("inDateRange(%v, %v, %v) = %v, want %v", tt.t, start, end, got, tt.want)
			}
		})
	}
}
