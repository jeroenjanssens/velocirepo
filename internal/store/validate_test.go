package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jeroenjanssens/velocirepo/internal/source"
)

func writeTestJSONL(t *testing.T, path string, records []source.Record) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for _, r := range records {
		data, _ := json.Marshal(r)
		f.Write(data)
		f.Write([]byte{'\n'})
	}
}

func writeTestRaw(t *testing.T, path string, lines []string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for _, line := range lines {
		f.WriteString(line + "\n")
	}
}

func writeTestEvents(t *testing.T, path string, events []source.GitHubEvent) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for _, e := range events {
		data, _ := json.Marshal(e)
		f.Write(data)
		f.Write([]byte{'\n'})
	}
}

func TestValidateData_Clean(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")

	writeTestJSONL(t, filepath.Join(dataDir, "pypi", "myproj", "2025-06-01.jsonl"), []source.Record{
		{Source: "pypi", Metric: "daily_downloads", ProjectID: "myproj", Target: "myproj", Date: "2025-06-01", Value: 100},
	})

	projects := map[string]bool{"myproj": true}
	result, err := ValidateData(dataDir, projects)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Issues) != 0 {
		t.Errorf("expected 0 issues, got %d: %v", len(result.Issues), result.Issues)
	}
	if result.FilesRead != 1 {
		t.Errorf("expected 1 file read, got %d", result.FilesRead)
	}
	if result.RecordCount != 1 {
		t.Errorf("expected 1 record, got %d", result.RecordCount)
	}
}

func TestValidateData_MalformedJSON(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")

	writeTestRaw(t, filepath.Join(dataDir, "pypi", "myproj", "2025-06-01.jsonl"), []string{
		`{"source":"pypi","metric":"daily_downloads","project_id":"myproj","target":"myproj","date":"2025-06-01","value":100}`,
		`{invalid json`,
		`{"source":"pypi","metric":"total_downloads","project_id":"myproj","target":"myproj","date":"2025-06-01","value":200}`,
	})

	projects := map[string]bool{"myproj": true}
	result, err := ValidateData(dataDir, projects)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(result.Issues))
	}
	if result.Issues[0].Type != IssueMalformedJSON {
		t.Errorf("expected malformed JSON issue, got %v", result.Issues[0].Type)
	}
	if result.Issues[0].Line != 2 {
		t.Errorf("expected line 2, got %d", result.Issues[0].Line)
	}
	if !result.Issues[0].Fixable {
		t.Error("expected issue to be fixable")
	}
}

func TestValidateData_InvalidDate(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")

	writeTestJSONL(t, filepath.Join(dataDir, "pypi", "myproj", "2025-06-01.jsonl"), []source.Record{
		{Source: "pypi", Metric: "daily_downloads", ProjectID: "myproj", Target: "myproj", Date: "2025-6-1", Value: 100},
	})

	projects := map[string]bool{"myproj": true}
	result, err := ValidateData(dataDir, projects)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(result.Issues))
	}
	if result.Issues[0].Type != IssueInvalidDate {
		t.Errorf("expected invalid date issue, got %v", result.Issues[0].Type)
	}
}

func TestValidateData_DateMismatch(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")

	writeTestJSONL(t, filepath.Join(dataDir, "pypi", "myproj", "2025-06-01.jsonl"), []source.Record{
		{Source: "pypi", Metric: "daily_downloads", ProjectID: "myproj", Target: "myproj", Date: "2025-06-01", Value: 100},
		{Source: "pypi", Metric: "daily_downloads", ProjectID: "myproj", Target: "myproj", Date: "2025-06-02", Value: 200},
	})

	projects := map[string]bool{"myproj": true}
	result, err := ValidateData(dataDir, projects)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(result.Issues))
	}
	if result.Issues[0].Type != IssueDateMismatch {
		t.Errorf("expected date mismatch issue, got %v", result.Issues[0].Type)
	}
	if !result.Issues[0].Fixable {
		t.Error("expected issue to be fixable")
	}
}

func TestValidateData_DateMismatch_Monthly(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")

	writeTestJSONL(t, filepath.Join(dataDir, "pypi", "myproj", "2025-06.jsonl"), []source.Record{
		{Source: "pypi", Metric: "daily_downloads", ProjectID: "myproj", Target: "myproj", Date: "2025-06-15", Value: 100},
		{Source: "pypi", Metric: "daily_downloads", ProjectID: "myproj", Target: "myproj", Date: "2025-07-01", Value: 200},
	})

	projects := map[string]bool{"myproj": true}
	result, err := ValidateData(dataDir, projects)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(result.Issues))
	}
	if result.Issues[0].Type != IssueDateMismatch {
		t.Errorf("expected date mismatch issue, got %v", result.Issues[0].Type)
	}
}

func TestValidateData_EmptyFields(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")

	writeTestJSONL(t, filepath.Join(dataDir, "pypi", "myproj", "2025-06-01.jsonl"), []source.Record{
		{Source: "pypi", Metric: "", ProjectID: "myproj", Target: "myproj", Date: "2025-06-01", Value: 100},
		{Source: "pypi", Metric: "daily_downloads", ProjectID: "", Target: "myproj", Date: "2025-06-01", Value: 200},
	})

	projects := map[string]bool{"myproj": true}
	result, err := ValidateData(dataDir, projects)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(result.Issues))
	}
	for _, issue := range result.Issues {
		if issue.Type != IssueEmptyField {
			t.Errorf("expected empty field issue, got %v", issue.Type)
		}
	}
}

func TestValidateData_Duplicates(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")

	writeTestJSONL(t, filepath.Join(dataDir, "pypi", "myproj", "2025-06-01.jsonl"), []source.Record{
		{Source: "pypi", Metric: "daily_downloads", ProjectID: "myproj", Target: "myproj", Date: "2025-06-01", Value: 100},
		{Source: "pypi", Metric: "daily_downloads", ProjectID: "myproj", Target: "myproj", Date: "2025-06-01", Value: 100},
		{Source: "pypi", Metric: "daily_downloads", ProjectID: "myproj", Target: "myproj", Date: "2025-06-01", Value: 200},
	})

	projects := map[string]bool{"myproj": true}
	result, err := ValidateData(dataDir, projects)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Issues) != 2 {
		t.Fatalf("expected 2 issues, got %d: %v", len(result.Issues), result.Issues)
	}
	for _, issue := range result.Issues {
		if issue.Type != IssueDuplicate {
			t.Errorf("expected duplicate issue, got %v", issue.Type)
		}
	}
}

func TestValidateData_OrphanDir(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")

	writeTestJSONL(t, filepath.Join(dataDir, "pypi", "unknown-proj", "2025-06-01.jsonl"), []source.Record{
		{Source: "pypi", Metric: "daily_downloads", ProjectID: "unknown-proj", Target: "x", Date: "2025-06-01", Value: 100},
	})

	projects := map[string]bool{"myproj": true}
	result, err := ValidateData(dataDir, projects)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(result.Issues))
	}
	if result.Issues[0].Type != IssueOrphanDir {
		t.Errorf("expected orphan dir issue, got %v", result.Issues[0].Type)
	}
	if !result.Issues[0].Fixable {
		t.Error("expected issue to be fixable")
	}
}

func TestValidateData_GitHubEvents(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")

	writeTestEvents(t, filepath.Join(dataDir, "github", "myproj", "2025-06-01.jsonl"), []source.GitHubEvent{
		{Source: "github", EventType: "star", ProjectID: "myproj", GitHubRepo: "owner/repo", Datetime: "2025-06-01T10:00:00Z", User: "alice"},
		{Source: "github", EventType: "star", ProjectID: "myproj", GitHubRepo: "owner/repo", Datetime: "2025-06-01T10:00:00Z", User: "alice"},
	})

	projects := map[string]bool{"myproj": true}
	result, err := ValidateData(dataDir, projects)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d: %v", len(result.Issues), result.Issues)
	}
	if result.Issues[0].Type != IssueDuplicate {
		t.Errorf("expected duplicate issue, got %v", result.Issues[0].Type)
	}
}

func TestValidateData_GitHubEvents_InvalidDatetime(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")

	writeTestEvents(t, filepath.Join(dataDir, "github", "myproj", "2025-06-01.jsonl"), []source.GitHubEvent{
		{Source: "github", EventType: "star", ProjectID: "myproj", GitHubRepo: "owner/repo", Datetime: "bad", User: "alice"},
	})

	projects := map[string]bool{"myproj": true}
	result, err := ValidateData(dataDir, projects)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(result.Issues))
	}
	if result.Issues[0].Type != IssueInvalidDatetime {
		t.Errorf("expected invalid datetime issue, got %v", result.Issues[0].Type)
	}
}

func TestValidateData_UnexpectedFile(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")

	writeTestRaw(t, filepath.Join(dataDir, "pypi", "myproj", "notes.txt"), []string{"hello"})

	projects := map[string]bool{"myproj": true}
	result, err := ValidateData(dataDir, projects)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(result.Issues))
	}
	if result.Issues[0].Type != IssueUnexpectedFile {
		t.Errorf("expected unexpected file issue, got %v", result.Issues[0].Type)
	}
}

func TestFixMalformedJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	writeTestRaw(t, path, []string{
		`{"source":"pypi","metric":"x","project_id":"p","date":"2025-01-01","value":1}`,
		`{bad`,
		`{"source":"pypi","metric":"y","project_id":"p","date":"2025-01-01","value":2}`,
	})

	result := FixMalformedJSON(map[string][]int{path: {2}})
	if result.Fixed != 1 {
		t.Errorf("expected 1 fixed, got %d", result.Fixed)
	}

	records, err := ReadRecords(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Errorf("expected 2 records after fix, got %d", len(records))
	}
}

func TestFixDuplicates(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	path := filepath.Join(dataDir, "pypi", "myproj", "2025-06-01.jsonl")

	writeTestJSONL(t, path, []source.Record{
		{Source: "pypi", Metric: "daily_downloads", ProjectID: "myproj", Target: "myproj", Date: "2025-06-01", Value: 100},
		{Source: "pypi", Metric: "daily_downloads", ProjectID: "myproj", Target: "myproj", Date: "2025-06-01", Value: 100},
	})

	result := FixDuplicates([]string{path})
	if result.Fixed != 1 {
		t.Errorf("expected 1 fixed, got %d", result.Fixed)
	}

	records, err := ReadRecords(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Errorf("expected 1 record after fix, got %d", len(records))
	}
}

func TestFixDateMismatches(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	path := filepath.Join(dataDir, "pypi", "myproj", "2025-06-01.jsonl")

	writeTestJSONL(t, path, []source.Record{
		{Source: "pypi", Metric: "daily_downloads", ProjectID: "myproj", Target: "myproj", Date: "2025-06-01", Value: 100},
		{Source: "pypi", Metric: "daily_downloads", ProjectID: "myproj", Target: "myproj", Date: "2025-06-02", Value: 200},
	})

	result := FixDateMismatches([]string{path})
	if result.Fixed != 1 {
		t.Errorf("expected 1 fixed, got %d", result.Fixed)
	}

	records, err := ReadRecords(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Errorf("expected 1 record in original file, got %d", len(records))
	}
	if records[0].Date != "2025-06-01" {
		t.Errorf("expected date 2025-06-01, got %s", records[0].Date)
	}

	movedPath := filepath.Join(dataDir, "pypi", "myproj", "2025-06-02.jsonl")
	movedRecords, err := ReadRecords(movedPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(movedRecords) != 1 {
		t.Errorf("expected 1 record in new file, got %d", len(movedRecords))
	}
	if movedRecords[0].Date != "2025-06-02" {
		t.Errorf("expected date 2025-06-02, got %s", movedRecords[0].Date)
	}
}

func TestFixOrphanDirs(t *testing.T) {
	dir := t.TempDir()
	orphanPath := filepath.Join(dir, "data", "pypi", "orphan")
	os.MkdirAll(orphanPath, 0755)

	result := FixOrphanDirs([]string{orphanPath})
	if result.Fixed != 1 {
		t.Errorf("expected 1 fixed, got %d", result.Fixed)
	}

	if _, err := os.Stat(orphanPath); !os.IsNotExist(err) {
		t.Error("expected orphan dir to be removed")
	}
}

func TestValidateData_NoProjectFilter(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")

	writeTestJSONL(t, filepath.Join(dataDir, "pypi", "anything", "2025-06-01.jsonl"), []source.Record{
		{Source: "pypi", Metric: "daily_downloads", ProjectID: "anything", Target: "x", Date: "2025-06-01", Value: 100},
	})

	result, err := ValidateData(dataDir, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Issues) != 0 {
		t.Errorf("expected 0 issues when no project filter, got %d", len(result.Issues))
	}
}

func TestDateMatchesFile(t *testing.T) {
	tests := []struct {
		date      string
		fileRange string
		want      bool
	}{
		{"2025-06-01", "2025-06-01", true},
		{"2025-06-02", "2025-06-01", false},
		{"2025-06-15", "2025-06", true},
		{"2025-07-01", "2025-06", false},
		{"2025-03-15", "2025", true},
		{"2026-01-01", "2025", false},
	}

	for _, tt := range tests {
		got := dateMatchesFile(tt.date, tt.fileRange)
		if got != tt.want {
			t.Errorf("dateMatchesFile(%q, %q) = %v, want %v", tt.date, tt.fileRange, got, tt.want)
		}
	}
}
