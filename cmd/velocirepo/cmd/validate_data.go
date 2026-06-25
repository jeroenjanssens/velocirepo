package cmd

import (
	"bufio"
	"fmt"
	"os"
	"sort"

	"github.com/jeroenjanssens/velocirepo/internal/store"
	"github.com/jeroenjanssens/velocirepo/internal/ui"
	"github.com/spf13/cobra"
)

func validateDataCmd() *cobra.Command {
	var fix bool

	cmd := &cobra.Command{
		Use:     "validate-data",
		Short:   "Check data files for integrity issues",
		GroupID: "data",
		RunE: func(cmd *cobra.Command, args []string) error {
			dataDir := cfg.DataDir()

			var projectIDs map[string]bool
			if projects := cfg.ResolveProjects(); projects != nil {
				projectIDs = make(map[string]bool, len(projects))
				for id := range projects {
					projectIDs[id] = true
				}
			}

			ui.Infof("scanning %s", dataDir)

			result, err := store.ValidateData(dataDir, projectIDs)
			if err != nil {
				return err
			}

			fmt.Fprintf(os.Stdout, "Scanned %d files, %d lines, %d records\n\n",
				result.FilesRead, result.LinesRead, result.RecordCount)

			if len(result.Issues) == 0 {
				fmt.Fprintln(os.Stdout, "No issues found.")
				return nil
			}

			grouped := groupIssuesByType(result.Issues)
			types := sortedIssueTypes(grouped)

			for _, t := range types {
				issues := grouped[t]
				fmt.Fprintf(os.Stdout, "%s (%d)\n", t, len(issues))
				for _, issue := range issues {
					marker := "  ✗"
					if issue.Fixable {
						marker = "  ⚠"
					}
					fmt.Fprintf(os.Stdout, "%s %s: %s\n", marker, relativePath(dataDir, issue.Path), issue.Message)
				}
				fmt.Fprintln(os.Stdout)
			}

			fixable := countFixable(result.Issues)
			fmt.Fprintf(os.Stdout, "%d issue(s), %d fixable\n", len(result.Issues), fixable)

			if fixable == 0 || !fix {
				if fixable > 0 && !fix {
					fmt.Fprintln(os.Stdout, "\nRun with --fix to apply fixes interactively.")
				}
				if len(result.Issues) > 0 {
					return fmt.Errorf("%d issue(s) found", len(result.Issues))
				}
				return nil
			}

			fmt.Fprintln(os.Stdout)

			if !isInteractive() {
				return fmt.Errorf("--fix requires an interactive terminal")
			}

			reader := bufio.NewReader(os.Stdin)

			malformedPaths := collectMalformedPaths(grouped[store.IssueMalformedJSON])
			if len(malformedPaths) > 0 {
				lineCount := 0
				for _, lines := range malformedPaths {
					lineCount += len(lines)
				}
				ok, err := confirm(os.Stdout, reader,
					fmt.Sprintf("Remove %d malformed JSON line(s) from %d file(s)?", lineCount, len(malformedPaths)))
				if err != nil {
					return err
				}
				if ok {
					r := store.FixMalformedJSON(malformedPaths)
					fmt.Fprintf(os.Stdout, "  Removed %d line(s)\n", r.Fixed)
					for _, e := range r.Errors {
						ui.Errorf("  %v", e)
					}
				}
			}

			dupPaths := collectDuplicatePaths(grouped[store.IssueDuplicate])
			if len(dupPaths) > 0 {
				dupCount := len(grouped[store.IssueDuplicate])
				ok, err := confirm(os.Stdout, reader,
					fmt.Sprintf("Deduplicate %d record(s) across %d file(s)?", dupCount, len(dupPaths)))
				if err != nil {
					return err
				}
				if ok {
					r := store.FixDuplicates(dupPaths)
					fmt.Fprintf(os.Stdout, "  Removed %d duplicate(s)\n", r.Fixed)
					for _, e := range r.Errors {
						ui.Errorf("  %v", e)
					}
				}
			}

			mismatchPaths := collectMismatchPaths(grouped[store.IssueDateMismatch])
			if len(mismatchPaths) > 0 {
				mismatchCount := len(grouped[store.IssueDateMismatch])
				ok, err := confirm(os.Stdout, reader,
					fmt.Sprintf("Move %d misplaced record(s) to correct files across %d file(s)?", mismatchCount, len(mismatchPaths)))
				if err != nil {
					return err
				}
				if ok {
					r := store.FixDateMismatches(mismatchPaths)
					fmt.Fprintf(os.Stdout, "  Moved %d record(s)\n", r.Fixed)
					for _, e := range r.Errors {
						ui.Errorf("  %v", e)
					}
				}
			}

			orphanPaths := collectOrphanPaths(grouped[store.IssueOrphanDir])
			if len(orphanPaths) > 0 {
				ok, err := confirm(os.Stdout, reader,
					fmt.Sprintf("Remove %d orphan director(ies) not in config?", len(orphanPaths)))
				if err != nil {
					return err
				}
				if ok {
					r := store.FixOrphanDirs(orphanPaths)
					fmt.Fprintf(os.Stdout, "  Removed %d director(ies)\n", r.Fixed)
					for _, e := range r.Errors {
						ui.Errorf("  %v", e)
					}
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&fix, "fix", false, "interactively fix issues")

	return cmd
}

func groupIssuesByType(issues []store.Issue) map[store.IssueType][]store.Issue {
	grouped := make(map[store.IssueType][]store.Issue)
	for _, issue := range issues {
		grouped[issue.Type] = append(grouped[issue.Type], issue)
	}
	return grouped
}

func sortedIssueTypes(grouped map[store.IssueType][]store.Issue) []store.IssueType {
	types := make([]store.IssueType, 0, len(grouped))
	for t := range grouped {
		types = append(types, t)
	}
	sort.Slice(types, func(i, j int) bool {
		return types[i] < types[j]
	})
	return types
}

func countFixable(issues []store.Issue) int {
	count := 0
	for _, issue := range issues {
		if issue.Fixable {
			count++
		}
	}
	return count
}

func relativePath(base, path string) string {
	if len(path) > len(base) && path[:len(base)] == base {
		return path[len(base)+1:]
	}
	return path
}

func collectMalformedPaths(issues []store.Issue) map[string][]int {
	if len(issues) == 0 {
		return nil
	}
	paths := make(map[string][]int)
	for _, issue := range issues {
		paths[issue.Path] = append(paths[issue.Path], issue.Line)
	}
	return paths
}

func collectDuplicatePaths(issues []store.Issue) []string {
	if len(issues) == 0 {
		return nil
	}
	seen := make(map[string]bool)
	var paths []string
	for _, issue := range issues {
		if !seen[issue.Path] {
			seen[issue.Path] = true
			paths = append(paths, issue.Path)
		}
	}
	return paths
}

func collectMismatchPaths(issues []store.Issue) []string {
	if len(issues) == 0 {
		return nil
	}
	seen := make(map[string]bool)
	var paths []string
	for _, issue := range issues {
		if !seen[issue.Path] {
			seen[issue.Path] = true
			paths = append(paths, issue.Path)
		}
	}
	return paths
}

func collectOrphanPaths(issues []store.Issue) []string {
	if len(issues) == 0 {
		return nil
	}
	paths := make([]string, 0, len(issues))
	for _, issue := range issues {
		paths = append(paths, issue.Path)
	}
	return paths
}
