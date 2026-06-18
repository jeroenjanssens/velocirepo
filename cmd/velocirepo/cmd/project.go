package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	"github.com/spf13/cobra"
)

var errAborted = errors.New("aborted")

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

func prompt(w io.Writer, r *bufio.Reader, label string, defaultVal string, source string) (string, error) {
	if defaultVal != "" {
		if source != "" {
			fmt.Fprintf(w, "%s [%s] (from %s): ", label, defaultVal, source)
		} else {
			fmt.Fprintf(w, "%s [%s]: ", label, defaultVal)
		}
	} else {
		fmt.Fprintf(w, "%s: ", label)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	defer signal.Stop(sigCh)

	inputCh := make(chan string, 1)
	errCh := make(chan error, 1)
	go func() {
		line, err := r.ReadString('\n')
		if err != nil {
			errCh <- err
		} else {
			inputCh <- line
		}
	}()

	select {
	case <-sigCh:
		fmt.Fprintln(w)
		return "", errAborted
	case err := <-errCh:
		_ = err
		fmt.Fprintln(w)
		return "", errAborted
	case input := <-inputCh:
		input = strings.TrimSpace(input)
		if input == "" {
			return defaultVal, nil
		}
		return input, nil
	}
}

// promptWithHint is like prompt but suppresses the default value when suppress is true.
// It returns the entered value and whether the suggestion was overridden.
func promptWithHint(w io.Writer, r *bufio.Reader, label string, defaultVal string, source string, suppress bool) (string, bool, error) {
	if suppress {
		defaultVal = ""
		source = ""
	}
	result, err := prompt(w, r, label, defaultVal, source)
	if err != nil {
		return "", false, err
	}
	overridden := !suppress && defaultVal != "" && result != defaultVal
	return result, overridden, nil
}

func confirm(w io.Writer, r *bufio.Reader, message string) (bool, error) {
	fmt.Fprintf(w, "%s [y/N]: ", message)
	input, err := r.ReadString('\n')
	if err != nil {
		fmt.Fprintln(w)
		return false, errAborted
	}
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes", nil
}
