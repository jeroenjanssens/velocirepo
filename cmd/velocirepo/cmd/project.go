package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func projectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "project",
		Aliases: []string{"proj"},
		Short:   "Manage projects in the config file",
	}

	cmd.AddCommand(projectInitCmd())
	cmd.AddCommand(projectListCmd())
	cmd.AddCommand(projectAddCmd())
	cmd.AddCommand(projectRemoveCmd())
	cmd.AddCommand(projectUpdateCmd())
	cmd.AddCommand(projectShowCmd())
	cmd.AddCommand(projectRenameCmd())
	cmd.AddCommand(projectValidateCmd())
	cmd.AddCommand(projectImportCmd())

	return cmd
}

func isInteractive() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func prompt(w io.Writer, r *bufio.Reader, label string, defaultVal string, source string) string {
	if defaultVal != "" {
		if source != "" {
			fmt.Fprintf(w, "%s [%s] (from %s): ", label, defaultVal, source)
		} else {
			fmt.Fprintf(w, "%s [%s]: ", label, defaultVal)
		}
	} else {
		fmt.Fprintf(w, "%s: ", label)
	}
	input, _ := r.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal
	}
	return input
}

func confirm(w io.Writer, r *bufio.Reader, message string) bool {
	fmt.Fprintf(w, "%s [y/N]: ", message)
	input, _ := r.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes"
}
