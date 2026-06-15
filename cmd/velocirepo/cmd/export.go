package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jeroenjanssens/velocirepo/internal/store"
	"github.com/spf13/cobra"
)

func exportCmd() *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export metrics data to a file",
		Long:  "Export all metrics data to Parquet or CSV format. The format is determined by the file extension.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ext := strings.ToLower(filepath.Ext(output))
			dataDir := cfg.DataDir()

			switch ext {
			case ".parquet":
				if err := store.ExportParquet(dataDir, output); err != nil {
					return err
				}
			case ".csv":
				if err := store.ExportCSV(dataDir, output); err != nil {
					return err
				}
			default:
				return fmt.Errorf("unsupported format %q (use .parquet or .csv)", ext)
			}

			abs, _ := filepath.Abs(output)
			info, err := os.Stat(abs)
			if err != nil {
				return nil
			}
			fmt.Fprintf(os.Stdout, "Exported metrics to %s (%s)\n", output, formatSize(info.Size()))
			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "output file path (.parquet or .csv)")
	cmd.MarkFlagRequired("output")

	return cmd
}

