package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jeroenjanssens/velocirepo/internal/store"
	"github.com/spf13/cobra"
)

func showProjectCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:     "show-project <id>",
		Short:   "Show detailed information about a project",
		Args:    cobra.ExactArgs(1),
		GroupID: "project",
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]

			projects := cfg.ResolveProjects()
			proj, exists := projects[id]
			if !exists {
				return fmt.Errorf("project %q not found in config", id)
			}

			dataDir := cfg.DataDir()
			sources := listSources(proj)

			type sourceStats struct {
				Source    string `json:"source"`
				LastDate  string `json:"last_date"`
				Records   int    `json:"records"`
				SizeBytes int64  `json:"size_bytes"`
			}

			var stats []sourceStats
			var totalRecords int
			var totalSize int64

			for _, src := range sources {
				dir := filepath.Join(dataDir, sourceDataDir(src), id)
				lastDate, records, size := scanSourceDir(dir)
				stats = append(stats, sourceStats{
					Source:    src,
					LastDate:  lastDate,
					Records:   records,
					SizeBytes: size,
				})
				totalRecords += records
				totalSize += size
			}

			if jsonOutput {
				out := struct {
					ID            string        `json:"id"`
					Name          string        `json:"name"`
					GitHubEvents  []string      `json:"github,omitempty"`
					GitHubTraffic []string      `json:"github-traffic,omitempty"`
					PyPI          []string      `json:"pypi,omitempty"`
					CRAN          []string      `json:"cran,omitempty"`
					Homebrew      []string      `json:"homebrew,omitempty"`
					Plausible     []string      `json:"plausible,omitempty"`
					OpenVSX       []string      `json:"openvsx,omitempty"`
					Sources       []sourceStats `json:"sources"`
					Total         struct {
						Records   int   `json:"records"`
						SizeBytes int64 `json:"size_bytes"`
					} `json:"total"`
				}{
					ID:            id,
					Name:          proj.Name,
					GitHubEvents:  []string(proj.GitHubEvents),
					GitHubTraffic: []string(proj.GitHubTraffic),
					PyPI:          []string(proj.PyPI),
					CRAN:          []string(proj.CRAN),
					Homebrew:      []string(proj.Homebrew),
					Plausible:     []string(proj.Plausible),
					OpenVSX:       []string(proj.OpenVSX),
					Sources:       stats,
				}
				out.Total.Records = totalRecords
				out.Total.SizeBytes = totalSize
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(out)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Project: %s\n", id)
			fmt.Fprintf(cmd.OutOrStdout(), "Name:    %s\n", proj.Name)
			if !proj.GitHubEvents.IsEmpty() {
				fmt.Fprintf(cmd.OutOrStdout(), "GitHub:  %s\n", proj.GitHubEvents.String())
			}
			if !proj.GitHubTraffic.IsEmpty() {
				fmt.Fprintf(cmd.OutOrStdout(), "GitHub Traffic: %s\n", proj.GitHubTraffic.String())
			}
			if !proj.PyPI.IsEmpty() {
				fmt.Fprintf(cmd.OutOrStdout(), "PyPI:    %s\n", proj.PyPI.String())
			}
			if !proj.CRAN.IsEmpty() {
				fmt.Fprintf(cmd.OutOrStdout(), "CRAN:    %s\n", proj.CRAN.String())
			}
			if !proj.Homebrew.IsEmpty() {
				fmt.Fprintf(cmd.OutOrStdout(), "Homebrew: %s\n", proj.Homebrew.String())
			}
			if !proj.Plausible.IsEmpty() {
				fmt.Fprintf(cmd.OutOrStdout(), "Plausible: %s\n", proj.Plausible.String())
			}
			if !proj.OpenVSX.IsEmpty() {
				fmt.Fprintf(cmd.OutOrStdout(), "OpenVSX: %s\n", proj.OpenVSX.String())
			}

			if len(stats) > 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "\nSources:")
				for _, s := range stats {
					lastStr := "never fetched"
					if s.LastDate != "" {
						lastStr = "last fetched: " + s.LastDate
					}
					fmt.Fprintf(cmd.OutOrStdout(), "  %-12s %s   records: %d   size: %s\n",
						s.Source, lastStr, s.Records, formatSize(s.SizeBytes))
				}
				fmt.Fprintf(cmd.OutOrStdout(), "\nTotal: %d records across %d sources (%s)\n",
					totalRecords, len(stats), formatSize(totalSize))
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output as JSON")

	return cmd
}


func scanSourceDir(dir string) (lastDate string, records int, size int64) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", 0, 0
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		size += info.Size()

		// Count records
		path := filepath.Join(dir, e.Name())
		recs, _ := store.ReadRecords(path)
		records += len(recs)

		// Track last date from filename
		datePart := strings.TrimSuffix(e.Name(), ".jsonl")
		if datePart > lastDate {
			lastDate = datePart
		}
	}
	return lastDate, records, size
}

func formatSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%d KB", bytes/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
}

