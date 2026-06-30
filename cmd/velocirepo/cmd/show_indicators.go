package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/jeroenjanssens/velocirepo/internal/store"
	"github.com/spf13/cobra"
)

func showIndicatorsCmd() *cobra.Command {
	var defaultsOnly bool

	cmd := &cobra.Command{
		Use:     "show-indicators",
		Short:   "Print indicator definitions as TOML",
		GroupID: "query",
		RunE: func(cmd *cobra.Command, args []string) error {
			var indicators []store.IndicatorDef
			if defaultsOnly {
				indicators = store.DefaultIndicators
			} else {
				indicators = indicatorDefs()
			}

			for i, ind := range indicators {
				if i > 0 {
					_, _ = fmt.Fprintln(os.Stdout)
				}
				_, _ = fmt.Fprintf(os.Stdout, "[indicators.%s]\n", ind.Name)
				_, _ = fmt.Fprintf(os.Stdout, "description = %q\n", ind.Description)
				_, _ = fmt.Fprintf(os.Stdout, "query = \"\"\"\n%s\n\"\"\"\n", strings.TrimSpace(ind.Query))
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&defaultsOnly, "defaults", false, "show only the built-in default indicators")

	return cmd
}
