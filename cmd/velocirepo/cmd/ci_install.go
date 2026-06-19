package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/jeroenjanssens/velocirepo/internal/version"
	"github.com/spf13/cobra"
)

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

      - uses: jeroenjanssens/velocirepo@{{.Version}}
        with:
          github-token: {{"{{"}} secrets.GH_TOKEN {{"}}"}}
{{- if .NeedsPlausible}}
          plausible-token: {{"{{"}} secrets.PLAUSIBLE_TOKEN {{"}}"}}
{{- end}}
{{- if .NeedsYouTube}}
          youtube-token: {{"{{"}} secrets.YOUTUBE_TOKEN {{"}}"}}
{{- end}}

      - name: Commit and push
        run: |
          git config --local user.email "github-actions[bot]@users.noreply.github.com"
          git config --local user.name "github-actions[bot]"
          git add velocirepo/
          git diff --staged --quiet || git commit -m "Update metrics - $(date -u +'%Y-%m-%d')"
          git push
`

func ciInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Generate GitHub Actions workflow for nightly metric fetching",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !isInteractive() {
				return fmt.Errorf("ci install requires an interactive terminal")
			}

			reader := bufio.NewReader(os.Stdin)

			defaultCron := "0 1 * * *"
			cron, err := prompt(os.Stdout, reader, "Cron schedule", defaultCron, "daily at 1am UTC")
			if err != nil {
				return err
			}

			defaultVersion := "latest"
			if version.Version != "dev" {
				defaultVersion = version.Version
			}
			versionHint := "use a tag like v0.1.4, or \"latest\""
			ver, err := prompt(os.Stdout, reader, "Velocirepo version", defaultVersion, versionHint)
			if err != nil {
				return err
			}

			defaultFile := ".github/workflows/velocirepo.yml"
			filename, err := prompt(os.Stdout, reader, "Workflow file", defaultFile, "")
			if err != nil {
				return err
			}

			outPath := filepath.Join(cfg.Dir, filename)

			if _, err := os.Stat(outPath); err == nil {
				ok, err := confirm(os.Stdout, reader, fmt.Sprintf("File %s already exists. Overwrite?", filename))
				if err != nil {
					return err
				}
				if !ok {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			needsPlausible := false
			needsYouTube := false
			projects := cfg.ResolveProjects()
			for _, p := range projects {
				if !p.Plausible.IsEmpty() {
					needsPlausible = true
				}
				if !p.YouTube.IsEmpty() {
					needsYouTube = true
				}
			}

			tmpl, err := template.New("workflow").Parse(workflowTemplate)
			if err != nil {
				return fmt.Errorf("parse template: %w", err)
			}

			outDir := filepath.Dir(outPath)
			if err := os.MkdirAll(outDir, 0755); err != nil {
				return fmt.Errorf("create directory: %w", err)
			}

			f, err := os.Create(outPath)
			if err != nil {
				return fmt.Errorf("create workflow file: %w", err)
			}
			defer f.Close()

			// Normalize version to a ref suitable for uses:
			actionRef := ver
			if !strings.HasPrefix(ver, "v") && ver != "latest" {
				actionRef = "v" + ver
			}
			if ver == "latest" {
				actionRef = "v0"
			}

			err = tmpl.Execute(f, struct {
				Cron           string
				Version        string
				NeedsPlausible bool
				NeedsYouTube   bool
			}{
				Cron:           cron,
				Version:        actionRef,
				NeedsPlausible: needsPlausible,
				NeedsYouTube:   needsYouTube,
			})
			if err != nil {
				return fmt.Errorf("render template: %w", err)
			}

			fmt.Printf("Created %s\n", outPath)
			return nil
		},
	}

	return cmd
}
