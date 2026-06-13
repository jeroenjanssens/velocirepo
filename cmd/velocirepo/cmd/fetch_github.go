package cmd

import (
	"github.com/jeroenjanssens/velocirepo/internal/config"
	"github.com/jeroenjanssens/velocirepo/internal/source"
	"github.com/spf13/cobra"
)

func fetchGitHubCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "github",
		Short: "Fetch GitHub events (stars, forks, issues, PRs, comments)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFetch(cmd, "github", func(id string, p config.Project) source.Source {
				if p.GitHub == "" {
					return nil
				}
				return &source.GitHub{
					Client: newHTTPClient(),
					Token:  githubToken(),
					Repo:   p.GitHub,
				}
			})
		},
	}
}
