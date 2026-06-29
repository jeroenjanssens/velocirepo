package mcp

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jeroenjanssens/velocirepo/internal/config"
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

func TestHandleUpdateProjectReloadsConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "velocirepo.toml")
	if err := os.WriteFile(cfgPath, []byte(`[projects.alpha]
name = "Alpha"
github = "org/alpha"
`), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	h := &handlers{cfg: cfg}

	result, err := h.handleUpdateProject(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{
			Arguments: map[string]any{
				"id":       "alpha",
				"linkedin": "urn:li:organization:123",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("handleUpdateProject returned error result: %#v", result.Content)
	}

	proj, err := h.cfg.GetProject("alpha")
	if err != nil {
		t.Fatal(err)
	}
	if got := []string(proj.LinkedIn); len(got) != 1 || got[0] != "urn:li:organization:123" {
		t.Fatalf("h.cfg LinkedIn = %#v, want urn:li:organization:123", got)
	}
}
