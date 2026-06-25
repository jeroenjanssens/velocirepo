package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/jeroenjanssens/velocirepo/internal/views"
	"github.com/spf13/cobra"
)

func setupViewsCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "setup-views",
		Short:   "Install dependencies for all views",
		Long:    "Run uv sync for Python views and renv::restore() for R views with renv.",
		GroupID: "view",
		RunE: func(cmd *cobra.Command, args []string) error {
			viewsDir := cfg.ViewsDir()
			allViews, err := views.Discover(viewsDir)
			if err != nil {
				return err
			}

			if len(allViews) == 0 {
				fmt.Println("No views found.")
				return nil
			}

			var setupCount int
			for _, v := range allViews {
				didSetup := false

				if _, err := os.Stat(filepath.Join(v.Dir, "pyproject.toml")); err == nil {
					fmt.Printf("Setting up '%s' (uv sync)...\n", v.Name)
					c := exec.Command("uv", "sync")
					c.Dir = v.Dir
					c.Stdout = os.Stdout
					c.Stderr = os.Stderr
					if err := c.Run(); err != nil {
						return fmt.Errorf("uv sync in %s: %w", v.Name, err)
					}
					didSetup = true
				}

				if _, err := os.Stat(filepath.Join(v.Dir, "renv.lock")); err == nil {
					fmt.Printf("Setting up '%s' (renv::restore)...\n", v.Name)
					c := exec.Command("Rscript", "-e", "renv::restore(prompt = FALSE)")
					c.Dir = v.Dir
					c.Stdout = os.Stdout
					c.Stderr = os.Stderr
					if err := c.Run(); err != nil {
						return fmt.Errorf("renv::restore in %s: %w", v.Name, err)
					}
					didSetup = true
				}

				if didSetup {
					setupCount++
				}
			}

			fmt.Printf("Set up %d view(s)\n", setupCount)
			return nil
		},
	}
}
