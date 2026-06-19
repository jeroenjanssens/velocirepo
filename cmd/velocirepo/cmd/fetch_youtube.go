package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/jeroenjanssens/velocirepo/internal/source"
	"github.com/jeroenjanssens/velocirepo/internal/store"
	"github.com/jeroenjanssens/velocirepo/internal/ui"
	"github.com/spf13/cobra"
)

func fetchYouTubeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "youtube",
		Short: "Fetch YouTube metrics (views, likes, comments, subscribers)",
		RunE: func(cmd *cobra.Command, args []string) error {
			projects := cfg.ResolveProjects()
			if projects == nil {
				return fmt.Errorf("no projects configured")
			}

			projects = filterProjects(projects)
			if projects == nil {
				return fmt.Errorf("project %q not found in config", fetchProject)
			}

			apiKey := youtubeAPIKey()
			if apiKey == "" {
				return fmt.Errorf("YOUTUBE_TOKEN not set")
			}

			endDate, err := resolveEndDate()
			if err != nil {
				return fmt.Errorf("parse end date: %w", err)
			}

			dataDir := cfg.DataDir()
			client := newHTTPClient()

			for id, proj := range projects {
				for _, target := range proj.YouTube {
					startDate, err := resolveStartDate(dataDir, "youtube", id)
					if err != nil {
						ui.Errorf("youtube/%s: resolve start date: %v", id, err)
						continue
					}

					if !startDate.Before(endDate.AddDate(0, 0, 1)) {
						ui.Skip("youtube", id, "up to date")
						continue
					}

					ui.Progress("youtube", id, startDate.Format("2006-01-02")+" → "+endDate.Format("2006-01-02"))

					src := &source.YouTube{Client: client, APIKey: apiKey, Target: target}
					records, err := src.Fetch(cmd.Context(), source.FetchOptions{
						ProjectID: id,
						StartDate: startDate,
						EndDate:   endDate,
					})
					if err != nil {
						ui.Errorf("youtube/%s: %v", id, err)
						continue
					}

					if len(records) == 0 {
						continue
					}

					if err := store.WriteRecords(dataDir, "youtube", id, records); err != nil {
						ui.Errorf("youtube/%s write: %v", id, err)
						continue
					}

					if entries := src.IndexEntries(); len(entries) > 0 {
						if err := store.WriteYouTubeIndex(dataDir, id, entries); err != nil {
							ui.Errorf("youtube/%s index: %v", id, err)
						}
					}

					ui.Done("youtube", id, len(records))
				}
			}

			if !noAggregate {
				if err := store.Aggregate(dataDir, time.Now().UTC()); err != nil {
					ui.Warnf("aggregation: %v", err)
				}
			}

			return nil
		},
	}
}

func youtubeAPIKey() string {
	return os.Getenv("YOUTUBE_TOKEN")
}
