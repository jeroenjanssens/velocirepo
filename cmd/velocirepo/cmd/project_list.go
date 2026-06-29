package cmd

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/jeroenjanssens/velocirepo/internal/config"
	"github.com/spf13/cobra"
)

func listProjectsCmd() *cobra.Command {
	var jsonOutput bool
	var quietOutput bool

	cmd := &cobra.Command{
		Use:     "list-projects",
		Short:   "List all configured projects",
		Aliases: []string{"ls-projects"},
		GroupID: "project",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			projects := cfg.Projects

			if quietOutput {
				for id := range projects {
					fmt.Fprintln(out, id)
				}
				return nil
			}

			if jsonOutput {
				type projectJSON struct {
					ID            string   `json:"id"`
					Name          string   `json:"name"`
					GitHubEvents  []string `json:"github,omitempty"`
					GitHubTraffic []string `json:"github-traffic,omitempty"`
					PyPI          []string `json:"pypi,omitempty"`
					CRAN          []string `json:"cran,omitempty"`
					Homebrew      []string `json:"homebrew,omitempty"`
					Plausible     []string `json:"plausible,omitempty"`
					OpenVSX       []string `json:"openvsx,omitempty"`
					YouTube       []string `json:"youtube,omitempty"`
					LinkedIn      []string `json:"linkedin,omitempty"`
				}
				var list []projectJSON
				for id, p := range projects {
					list = append(list, projectJSON{
						ID:            id,
						Name:          p.Name,
						GitHubEvents:  []string(p.GitHubEvents),
						GitHubTraffic: []string(p.GitHubTraffic),
						PyPI:          []string(p.PyPI),
						CRAN:          []string(p.CRAN),
						Homebrew:      []string(p.Homebrew),
						Plausible:     []string(p.Plausible),
						OpenVSX:       []string(p.OpenVSX),
						YouTube:       []string(p.YouTube),
						LinkedIn:      []string(p.LinkedIn),
					})
				}
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				return enc.Encode(list)
			}

			if len(projects) == 0 {
				fmt.Fprintln(out, "No projects configured.")
				return nil
			}

			w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tGITHUB-EVENTS\tGITHUB-TRAFFIC\tPYPI\tCRAN\tHOMEBREW\tPLAUSIBLE\tOPENVSX\tYOUTUBE\tLINKEDIN")
			for id, p := range projects {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
					id,
					valOrDash(p.Name),
					sliceOrDash(p.GitHubEvents),
					sliceOrDash(p.GitHubTraffic),
					sliceOrDash(p.PyPI),
					sliceOrDash(p.CRAN),
					sliceOrDash(p.Homebrew),
					sliceOrDash(p.Plausible),
					sliceOrDash(p.OpenVSX),
					sliceOrDash(p.YouTube),
					sliceOrDash(p.LinkedIn),
				)
			}
			return w.Flush()
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output as JSON")
	cmd.Flags().BoolVar(&quietOutput, "ids-only", false, "output only project IDs")

	return cmd
}

func valOrDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func sliceOrDash(s config.StringList) string {
	if s.IsEmpty() {
		return "—"
	}
	return s.String()
}
