package cmd

import (
	"strings"
	"testing"
)

// These commands previously wrote directly to os.Stdout, which bypassed
// cmd.OutOrStdout() and made their output invisible to the command's
// configured writer. They now route through cmd.OutOrStdout(); this test
// guards that by asserting the output reaches execCmd's buffer.

func TestVersionWritesToCommandOutput(t *testing.T) {
	cfgPath := setupTestConfig(t, `[projects.alpha]
name = "Alpha"
github = "org/alpha"
`)

	_, buf, err := execCmd(cfgPath, "version")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "velocirepo") {
		t.Errorf("version output not captured from command writer: %q", buf.String())
	}
}

func TestShowIndicatorsWritesToCommandOutput(t *testing.T) {
	cfgPath := setupTestConfig(t, `[projects.alpha]
name = "Alpha"
github = "org/alpha"
`)

	_, buf, err := execCmd(cfgPath, "show-indicators", "--defaults")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "[indicators.") {
		t.Errorf("show-indicators output not captured from command writer: %q", buf.String())
	}
}
