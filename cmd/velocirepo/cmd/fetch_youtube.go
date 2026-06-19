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
						ui.FetchError("youtube", id, fmt.Errorf("resolve start date: %w", err))
						continue
					}

					if !startDate.Before(endDate.AddDate(0, 0, 1)) {
						ui.FetchSkip("youtube", id, "already up to date")
						continue
					}

					dateRange := fmt.Sprintf("%s to %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
					ui.FetchStart("youtube", id, dateRange)
					started := time.Now()

					src := &source.YouTube{Client: client, APIKey: apiKey, Target: target}
					records, err := src.Fetch(cmd.Context(), source.FetchOptions{
						ProjectID: id,
						StartDate: startDate,
						EndDate:   endDate,
					})
					if err != nil {
						ui.FetchError("youtube", id, err)
						continue
					}

					if len(records) == 0 {
						ui.FetchSkip("youtube", id, "no new records")
						continue
					}

					if err := store.WriteRecords(dataDir, "youtube", id, records); err != nil {
						ui.FetchError("youtube", id, fmt.Errorf("write: %w", err))
						continue
					}

					if entries := src.IndexEntries(); len(entries) > 0 {
						if err := store.WriteYouTubeIndex(dataDir, id, entries); err != nil {
							ui.FetchWarn("youtube", id, fmt.Sprintf("index write: %v", err))
						}
					}

					ui.FetchDone("youtube", id, len(records), time.Since(started))
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
