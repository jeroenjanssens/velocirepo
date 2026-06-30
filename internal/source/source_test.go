package source

import (
	"encoding/json"
	"testing"
)

func TestRecordUnmarshalFloat(t *testing.T) {
	input := `{"metric":"rating","project_id":"p","date":"2026-01-01","value":5.0,"source":"openvsx","target":"ns/ext"}`
	var r Record
	if err := json.Unmarshal([]byte(input), &r); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if r.Value != 5 {
		t.Errorf("expected Value=5, got %d", r.Value)
	}
	if r.Metric != "rating" {
		t.Errorf("expected metric=rating, got %s", r.Metric)
	}
}

func TestRecordUnmarshalInt(t *testing.T) {
	input := `{"metric":"total_downloads","project_id":"p","date":"2026-01-01","value":1000,"source":"openvsx","target":"ns/ext"}`
	var r Record
	if err := json.Unmarshal([]byte(input), &r); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if r.Value != 1000 {
		t.Errorf("expected Value=1000, got %d", r.Value)
	}
}

func TestRecordUnmarshalRoundsFloat(t *testing.T) {
	input := `{"metric":"rating","project_id":"p","date":"2026-01-01","value":4.7,"source":"openvsx","target":"ns/ext"}`
	var r Record
	if err := json.Unmarshal([]byte(input), &r); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if r.Value != 5 {
		t.Errorf("expected Value=5 (rounded from 4.7), got %d", r.Value)
	}
}

func TestRecordUnmarshalRejectsUnrepresentableValue(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "parse overflow", value: "1e1000"},
		{name: "positive int64 overflow", value: "1e20"},
		{name: "negative int64 overflow", value: "-1e20"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := `{"metric":"total_downloads","project_id":"p","date":"2026-01-01","value":` + tt.value + `,"source":"openvsx","target":"ns/ext"}`
			var r Record
			if err := json.Unmarshal([]byte(input), &r); err == nil {
				t.Fatalf("expected unmarshal to fail for value %s", tt.value)
			}
		})
	}
}
