package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/posit-dev/velocirepo/internal/config"
	"github.com/posit-dev/velocirepo/internal/dateutil"
	"github.com/posit-dev/velocirepo/internal/store"
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

			proj, err := cfg.GetProject(id)
			if err != nil {
				return err
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
				dir := filepath.Join(dataDir, config.SourceDirPath(src), id)
				ds := store.ScanProjectDir(dir)

				// Report the last date the source was successfully fetched, not
				// the last date a value changed. For total_* gauges the two
				// differ: an unchanged total writes no row, so the newest
				// filename (ds.LastDate) can lag well behind the real fetch.
				// store.LastDate consults the watermark to close that gap.
				lastDate := ds.LastDate
				if t, err := store.LastDate(dataDir, src, id); err == nil && !t.IsZero() {
					lastDate = dateutil.FormatDate(t)
				}

				stats = append(stats, sourceStats{
					Source:    src,
					LastDate:  lastDate,
					Records:   ds.Records,
					SizeBytes: ds.Size,
				})
				totalRecords += ds.Records
				totalSize += ds.Size
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
					YouTube       []string      `json:"youtube,omitempty"`
					LinkedIn      []string      `json:"linkedin,omitempty"`
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
					YouTube:       []string(proj.YouTube),
					LinkedIn:      []string(proj.LinkedIn),
					Sources:       stats,
				}
				out.Total.Records = totalRecords
				out.Total.SizeBytes = totalSize
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(out)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Project: %s\n", id)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Name:    %s\n", proj.Name)
			for _, src := range proj.Sources() {
				if !src.Values.IsEmpty() {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-15s %s\n", src.DisplayName+":", src.Values.String())
				}
			}

			if len(stats) > 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\nSources:")
				for _, s := range stats {
					lastStr := "never fetched"
					if s.LastDate != "" {
						lastStr = "last fetched: " + s.LastDate
					}
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %-12s %s   records: %d   size: %s\n",
						s.Source, lastStr, s.Records, formatSize(s.SizeBytes))
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nTotal: %d records across %d sources (%s)\n",
					totalRecords, len(stats), formatSize(totalSize))
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output as JSON")

	return cmd
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
