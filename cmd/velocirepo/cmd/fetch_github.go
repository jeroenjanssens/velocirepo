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
			return runFetchMulti(cmd, "github", func(id string, p config.Project) []source.Source {
				var sources []source.Source
				for _, repo := range p.GitHub {
					sources = append(sources, &source.GitHub{
						Client: newHTTPClient(),
						Token:  githubToken(),
						Repo:   repo,
					})
				}
				return sources
			})
		},
	}
}
