package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/jeroenjanssens/velocirepo/internal/store"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
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
	var jsonFlag, csvFlag bool

	cmd := &cobra.Command{
		Use:   "run <sql>",
		Short: "Run an arbitrary SQL query",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			results, err := store.QueryLive(cfg.DataDir(), projectInfos(), args[0])
			if err != nil {
				return err
			}

			switch {
			case jsonFlag:
				return printJSON(results)
			case csvFlag:
				return printCSV(results)
			default:
				return printTable(results)
			}
		},
	}

	cmd.Flags().BoolVar(&jsonFlag, "json", false, "output as JSON")
	cmd.Flags().BoolVar(&csvFlag, "csv", false, "output as CSV")

	return cmd
}

func querySchemaCmd() *cobra.Command {
	var jsonFlag, csvFlag bool

	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Show table schemas",
		RunE: func(cmd *cobra.Command, args []string) error {
			cols, err := store.SchemaLive(cfg.DataDir(), projectInfos())
			if err != nil {
				return err
			}

			switch {
			case jsonFlag:
				results := make([]map[string]interface{}, len(cols))
				for i, c := range cols {
					results[i] = map[string]interface{}{
						"table":  c.Table,
						"column": c.Column,
						"type":   c.Type,
					}
				}
				return printJSON(results)
			case csvFlag:
				w := csv.NewWriter(os.Stdout)
				w.Write([]string{"table", "column", "type"})
				for _, c := range cols {
					w.Write([]string{c.Table, c.Column, c.Type})
				}
				w.Flush()
				return w.Error()
			default:
				table := tablewriter.NewTable(os.Stdout,
					tablewriter.WithHeaderAutoFormat(tw.Off),
					tablewriter.WithHeaderAlignment(tw.AlignLeft),
					tablewriter.WithRowAlignment(tw.AlignLeft),
					tablewriter.WithRendition(tw.Rendition{
						Symbols: tw.NewSymbols(tw.StyleLight),
					}),
				)
				table.Header([]string{"TABLE", "COLUMN", "TYPE"})
				for _, c := range cols {
					table.Append([]string{c.Table, c.Column, c.Type})
				}
				table.Render()
				return nil
			}
		},
	}

	cmd.Flags().BoolVar(&jsonFlag, "json", false, "output as JSON")
	cmd.Flags().BoolVar(&csvFlag, "csv", false, "output as CSV")

	return cmd
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

	table := tablewriter.NewTable(os.Stdout,
		tablewriter.WithHeaderAutoFormat(tw.Off),
		tablewriter.WithRendition(tw.Rendition{
			Symbols: tw.NewSymbols(tw.StyleLight),
		}),
	)
	table.Header(cols)

	for _, row := range results {
		vals := make([]string, len(cols))
		for i, col := range cols {
			vals[i] = formatValue(row[col])
		}
		table.Append(vals)
	}

	table.Render()
	return nil
}

func printJSON(results []map[string]interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(results)
}

func printCSV(results []map[string]interface{}) error {
	if len(results) == 0 {
		return nil
	}

	var cols []string
	for k := range results[0] {
		cols = append(cols, k)
	}
	sortColumns(cols)

	w := csv.NewWriter(os.Stdout)
	w.Write(cols)
	for _, row := range results {
		vals := make([]string, len(cols))
		for i, col := range cols {
			vals[i] = formatValue(row[col])
		}
		w.Write(vals)
	}
	w.Flush()
	return w.Error()
}

func formatValue(v interface{}) string {
	switch val := v.(type) {
	case time.Time:
		if val.Hour() == 0 && val.Minute() == 0 && val.Second() == 0 && val.Nanosecond() == 0 {
			return val.Format("2006-01-02")
		}
		return val.Format("2006-01-02T15:04:05Z")
	case nil:
		return "<null>"
	default:
		return fmt.Sprintf("%v", v)
	}
}

func projectInfos() []store.ProjectInfo {
	projects := cfg.ResolveProjects()
	if projects == nil {
		return nil
	}
	infos := make([]store.ProjectInfo, 0, len(projects))
	for id, p := range projects {
		infos = append(infos, store.ProjectInfo{
			ID:          id,
			Name:        p.Name,
			Description: p.Description,
			Color:       p.Color,
			Tags:        []string(p.Tags),
			Website:     p.Website,
			Logo:        p.Logo,
		})
	}
	return infos
}

func sortColumns(cols []string) {
	order := map[string]int{
		"project":     0,
		"source":      1,
		"metric":      2,
		"date":        3,
		"value":       4,
		"tags":        5,
		"event_type":  6,
		"github_repo": 7,
		"datetime":    8,
		"user":        9,
	}
	sort.Slice(cols, func(i, j int) bool {
		oi, oki := order[cols[i]]
		oj, okj := order[cols[j]]
		if !oki {
			oi = 100
		}
		if !okj {
			oj = 100
		}
		if oi != oj {
			return oi < oj
		}
		return cols[i] < cols[j]
	})
}
