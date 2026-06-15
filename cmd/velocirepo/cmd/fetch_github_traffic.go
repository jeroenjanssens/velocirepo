package cmd

import (
	"github.com/jeroenjanssens/velocirepo/internal/config"
	"github.com/jeroenjanssens/velocirepo/internal/source"
	"github.com/spf13/cobra"
)

func fetchGitHubTrafficCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "github-traffic",
		Short: "Fetch GitHub traffic data (views and clones)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFetchMulti(cmd, "github-traffic", func(id string, p config.Project) []source.Source {
				var sources []source.Source
				for _, repo := range p.GitHubTraffic {
					sources = append(sources, &source.GitHubTraffic{
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
