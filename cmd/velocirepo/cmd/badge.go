package cmd

import (
	"fmt"
	"os"

	"github.com/jeroenjanssens/velocirepo/internal/badge"
	"github.com/jeroenjanssens/velocirepo/internal/store"
	"github.com/spf13/cobra"
)

var badgePresets = map[string]struct {
	label string
	query string
	color string
}{
	"stars":     {"stars", "SELECT COUNT(*) AS value FROM github_events WHERE event_type = 'star'", "#007ec6"},
	"forks":     {"forks", "SELECT COUNT(*) AS value FROM github_events WHERE event_type = 'fork'", "#007ec6"},
	"downloads": {"downloads", "SELECT MAX(value) AS value FROM metrics WHERE metric = 'downloads' OR metric = 'total_downloads'", "#44cc11"},
	"pageviews": {"pageviews", "SELECT SUM(value) AS value FROM metrics WHERE metric = 'pageviews'", "#44cc11"},
}

func badgeCmd() *cobra.Command {
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
		Use:   "badge <type>",
		Short: "Generate SVG badges from metrics",
		Long: `Generate shields.io-style SVG badges. Available types: stars, forks, downloads, pageviews, custom.

For custom badges, provide --query and --label.`,
		Args:    cobra.ExactArgs(1),
		GroupID: "badge",
		ValidArgs: []string{"stars", "forks", "downloads", "pageviews", "custom"},
		RunE: func(cmd *cobra.Command, args []string) error {
			badgeType := args[0]

			if badgeType == "custom" {
				if query == "" {
					return fmt.Errorf("--query is required for custom badges")
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
			}

			preset, ok := badgePresets[badgeType]
			if !ok {
				return fmt.Errorf("unknown badge type %q (available: stars, forks, downloads, pageviews, custom)", badgeType)
			}

			q := preset.query
			if project != "" {
				q += fmt.Sprintf(" AND project = '%s'", project)
			}

			msg, err := queryBadgeValue(q)
			if err != nil {
				return err
			}

			if label == "" {
				label = preset.label
			}
			if color == "" {
				color = preset.color
			}

			return renderBadge(label, msg, color, labelColor, style, height, radius, output)
		},
	}

	cmd.Flags().StringVar(&query, "query", "", "SQL query returning a single value (for custom type)")
	cmd.Flags().StringVar(&project, "project", "", "scope to a specific project")
	cmd.Flags().StringVarP(&output, "output", "o", "", "output file (default: stdout)")
	cmd.Flags().StringVar(&style, "style", "flat", "badge style: flat, flat-square, plastic")
	cmd.Flags().StringVar(&color, "color", "", "message background color")
	cmd.Flags().StringVar(&labelColor, "label-color", "#555", "label background color")
	cmd.Flags().StringVar(&label, "label", "", "override label text")
	cmd.Flags().IntVar(&height, "height", 0, "badge height in pixels (0 = style default)")
	cmd.Flags().IntVar(&radius, "radius", -1, "corner radius (-1 = style default)")

	return cmd
}

func queryBadgeValue(query string) (string, error) {
	results, _, err := store.QueryLive(cfg.DataDir(), projectInfos(), indicatorDefs(), query)
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
