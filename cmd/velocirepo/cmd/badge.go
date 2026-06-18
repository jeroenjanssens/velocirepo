package cmd

import (
	"fmt"
	"os"

	"github.com/jeroenjanssens/velocirepo/internal/badge"
	"github.com/jeroenjanssens/velocirepo/internal/store"
	"github.com/spf13/cobra"
)

func badgeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "badge",
		Short: "Generate SVG badges from metrics",
	}

	cmd.AddCommand(badgePresetCmd("stars", "stars", "SELECT COUNT(*) AS value FROM github_events WHERE event_type = 'star'", "#007ec6"))
	cmd.AddCommand(badgePresetCmd("forks", "forks", "SELECT COUNT(*) AS value FROM github_events WHERE event_type = 'fork'", "#007ec6"))
	cmd.AddCommand(badgePresetCmd("downloads", "downloads", "SELECT MAX(value) AS value FROM metrics WHERE metric = 'downloads' OR metric = 'total_downloads'", "#44cc11"))
	cmd.AddCommand(badgePresetCmd("pageviews", "pageviews", "SELECT SUM(value) AS value FROM metrics WHERE metric = 'pageviews'", "#44cc11"))
	cmd.AddCommand(badgeCustomCmd())

	return cmd
}

func badgePresetCmd(use, defaultLabel, baseQuery, defaultColor string) *cobra.Command {
	var (
		project    string
		output     string
		style      string
		color      string
		labelColor string
		label      string
		height     int
		radius     int
	)

	cmd := &cobra.Command{
		Use:   use,
		Short: fmt.Sprintf("Generate a %s badge", use),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			query := baseQuery
			if project != "" {
				query += fmt.Sprintf(" AND project = '%s'", project)
			}

			msg, err := queryBadgeValue(query)
			if err != nil {
				return err
			}

			if label == "" {
				label = defaultLabel
			}
			if color == "" {
				color = defaultColor
			}

			return renderBadge(label, msg, color, labelColor, style, height, radius, output)
		},
	}

	addBadgeFlags(cmd, &project, &output, &style, &color, &labelColor, &label, &height, &radius)
	return cmd
}

func badgeCustomCmd() *cobra.Command {
	var (
		project    string
		output     string
		style      string
		color      string
		labelColor string
		label      string
		height     int
		radius     int
		query      string
	)

	cmd := &cobra.Command{
		Use:   "custom",
		Short: "Generate a badge from a custom query",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if query == "" {
				return fmt.Errorf("--query is required")
			}
			if label == "" {
				return fmt.Errorf("--label is required for custom badges")
			}

			q := query
			if project != "" {
				q = fmt.Sprintf("SELECT * FROM (%s) WHERE project = '%s'", q, project)
			}

			msg, err := queryBadgeValue(q)
			if err != nil {
				return err
			}

			if color == "" {
				color = "#007ec6"
			}

			return renderBadge(label, msg, color, labelColor, style, height, radius, output)
		},
	}

	cmd.Flags().StringVar(&query, "query", "", "SQL query returning a single value")
	addBadgeFlags(cmd, &project, &output, &style, &color, &labelColor, &label, &height, &radius)
	return cmd
}

func addBadgeFlags(cmd *cobra.Command, project, output, style, color, labelColor, label *string, height, radius *int) {
	cmd.Flags().StringVar(project, "project", "", "scope to a specific project")
	cmd.Flags().StringVarP(output, "output", "o", "", "output file (default: stdout)")
	cmd.Flags().StringVar(style, "style", "flat", "badge style: flat, flat-square, plastic")
	cmd.Flags().StringVar(color, "color", "", "message background color")
	cmd.Flags().StringVar(labelColor, "label-color", "#555", "label background color")
	cmd.Flags().StringVar(label, "label", "", "override label text")
	cmd.Flags().IntVar(height, "height", 0, "badge height in pixels (0 = style default)")
	cmd.Flags().IntVar(radius, "radius", -1, "corner radius (-1 = style default)")
}

func queryBadgeValue(query string) (string, error) {
	results, _, err := store.QueryLive(cfg.DataDir(), projectInfos(), query)
	if err != nil {
		return "", fmt.Errorf("badge query: %w", err)
	}
	if len(results) == 0 {
		return "0", nil
	}

	for _, v := range results[0] {
		switch val := v.(type) {
		case int64:
			return badge.FormatNumber(val), nil
		case float64:
			return badge.FormatNumber(int64(val)), nil
		case string:
			return val, nil
		default:
			return fmt.Sprintf("%v", v), nil
		}
	}
	return "0", nil
}

func renderBadge(label, msg, color, labelColor, style string, height, radius int, output string) error {
	opts := badge.Options{
		Label:      label,
		Message:    msg,
		Color:      color,
		LabelColor: labelColor,
		Style:      badge.Style(style),
		Height:     height,
		Radius:     radius,
	}

	svg := badge.Render(opts)

	if output == "" {
		fmt.Print(svg)
		return nil
	}

	if err := os.WriteFile(output, []byte(svg), 0644); err != nil {
		return fmt.Errorf("write badge: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Badge written to %s\n", output)
	return nil
}
