package source

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func fetchOptions(project string, start, end time.Time) FetchOptions {
	return FetchOptions{
		ProjectID: project,
		StartDate: start,
		EndDate:   end,
	}
}

func juneFetchOptions(project string, startDay, endDay int) FetchOptions {
	return fetchOptions(project, dateUTC(2025, 6, startDay), dateUTC(2025, 6, endDay))
}

func dateUTC(year, month, day int) time.Time {
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

func filterByMetric(records []Record, metric string) []Record {
	var filtered []Record
	for _, r := range records {
		if r.Metric == metric {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func assertRecordCount(t *testing.T, records []Record, want int) {
	t.Helper()
	if len(records) != want {
		t.Fatalf("got %d records, want %d", len(records), want)
	}
}

func assertMetricValues(t *testing.T, records []Record, want map[string]int64) {
	t.Helper()
	got := make(map[string]int64, len(records))
	for _, r := range records {
		got[r.Metric] = r.Value
	}
	for metric, wantValue := range want {
		if got[metric] != wantValue {
			t.Errorf("%s = %d, want %d", metric, got[metric], wantValue)
		}
	}
	for metric, gotValue := range got {
		if _, ok := want[metric]; !ok {
			t.Errorf("unexpected metric %s = %d", metric, gotValue)
		}
	}
}

func jsonHandler(t *testing.T, status int, payload any) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		if payload == nil {
			return
		}
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}
}

func errorServer(t *testing.T, status int) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
	}))
	t.Cleanup(server.Close)
	return server
}

func assertBearerToken(t *testing.T, r *http.Request, token string) {
	t.Helper()
	if got := r.Header.Get("Authorization"); got != "Bearer "+token {
		t.Errorf("Authorization = %q, want %q", got, "Bearer "+token)
	}
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
