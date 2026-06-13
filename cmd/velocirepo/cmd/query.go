package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/jeroenjanssens/velocirepo/internal/store"
	"github.com/spf13/cobra"
)

func queryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query",
		Short: "Query metrics data",
	}

	cmd.AddCommand(queryRunCmd())
	cmd.AddCommand(querySchemaCmd())

	return cmd
}

func queryRunCmd() *cobra.Command {
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "run [sql]",
		Short: "Run an arbitrary SQL query",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			results, err := store.QueryLive(cfg.DataDir(), args[0])
			if err != nil {
				return err
			}

			switch outputFormat {
			case "json":
				return printJSON(results)
			default:
				return printTable(results)
			}
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "format", "f", "table", "output format: table, json")

	return cmd
}

func querySchemaCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "schema",
		Short: "Show the metrics table schema",
		RunE: func(cmd *cobra.Command, args []string) error {
			cols, err := store.SchemaLive(cfg.DataDir())
			if err != nil {
				return err
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "TABLE\tCOLUMN\tTYPE\tNULLABLE\n")
			for _, c := range cols {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", c.Table, c.Column, c.Type, c.Nullable)
			}
			return w.Flush()
		},
	}
}

func printTable(results []map[string]interface{}) error {
	if len(results) == 0 {
		fmt.Println("(0 rows)")
		return nil
	}

	var cols []string
	for k := range results[0] {
		cols = append(cols, k)
	}
	sortColumns(cols)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, strings.Join(cols, "\t"))

	for _, row := range results {
		vals := make([]string, len(cols))
		for i, col := range cols {
			vals[i] = fmt.Sprintf("%v", row[col])
		}
		fmt.Fprintln(w, strings.Join(vals, "\t"))
	}
	return w.Flush()
}

func printJSON(results []map[string]interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(results)
}

func sortColumns(cols []string) {
	order := map[string]int{
		"project": 0,
		"source":  1,
		"metric":  2,
		"date":    3,
		"value":   4,
		"tags":    5,
	}
	for i := 0; i < len(cols); i++ {
		for j := i + 1; j < len(cols); j++ {
			oi, oki := order[cols[i]]
			oj, okj := order[cols[j]]
			if !oki {
				oi = 100
			}
			if !okj {
				oj = 100
			}
			if oi > oj || (oi == oj && cols[i] > cols[j]) {
				cols[i], cols[j] = cols[j], cols[i]
			}
		}
	}
}
