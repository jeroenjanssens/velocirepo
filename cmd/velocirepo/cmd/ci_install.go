package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/spf13/cobra"
)

var ciCron string

const workflowTemplate = `name: Fetch Metrics

on:
  schedule:
    - cron: '{{.Cron}}'
  workflow_dispatch:

permissions:
  contents: write

jobs:
  fetch:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: jeroenjanssens/velocirepo@v1
        with:
          github-token: {{"{{"}} secrets.GH_TOKEN {{"}}"}}
{{- if .NeedsPlausible}}
          plausible-key: {{"{{"}} secrets.PLAUSIBLE_KEY {{"}}"}}
{{- end}}

      - name: Commit and push
        run: |
          git config --local user.email "github-actions[bot]@users.noreply.github.com"
          git config --local user.name "github-actions[bot]"
          git add data/
          git diff --staged --quiet || git commit -m "Update metrics - $(date -u +'%Y-%m-%d')"
          git push
`

func ciInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Generate GitHub Actions workflow for nightly metric fetching",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := cfg.Dir
			workflowDir := filepath.Join(dir, ".github", "workflows")
			if err := os.MkdirAll(workflowDir, 0755); err != nil {
				return fmt.Errorf("create workflow directory: %w", err)
			}

			needsPlausible := false
			projects := cfg.ResolveProjects()
			for _, p := range projects {
				if p.Plausible != "" {
					needsPlausible = true
					break
				}
			}

			tmpl, err := template.New("workflow").Parse(workflowTemplate)
			if err != nil {
				return fmt.Errorf("parse template: %w", err)
			}

			outPath := filepath.Join(workflowDir, "velocirepo.yml")
			f, err := os.Create(outPath)
			if err != nil {
				return fmt.Errorf("create workflow file: %w", err)
			}
			defer f.Close()

			err = tmpl.Execute(f, struct {
				Cron           string
				NeedsPlausible bool
			}{
				Cron:           ciCron,
				NeedsPlausible: needsPlausible,
			})
			if err != nil {
				return fmt.Errorf("render template: %w", err)
			}

			fmt.Printf("Created %s\n", outPath)
			return nil
		},
	}

	cmd.Flags().StringVar(&ciCron, "cron", "0 1 * * *", "cron schedule (default: 1am UTC daily)")

	return cmd
}
