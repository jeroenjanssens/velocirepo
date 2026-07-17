package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/posit-dev/velocirepo/internal/testutil"
	"github.com/spf13/cobra"
)

func setupTestConfig(t *testing.T, content string) string {
	t.Helper()
	return testutil.WriteConfig(t, content)
}

func writeEnvFile(t *testing.T, dir, content string) string {
	t.Helper()
	return testutil.WriteTempFile(t, dir, ".env", content)
}

func writeTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	return testutil.WriteTempFile(t, dir, name, content)
}

func setupConfigAndEnv(t *testing.T, configContent, envContent string) (cfgPath, envPath string) {
	t.Helper()
	dir := t.TempDir()
	cfgPath = testutil.WriteTempFile(t, dir, "velocirepo.toml", configContent)
	envPath = writeEnvFile(t, dir, envContent)
	return cfgPath, envPath
}

func execCmd(cfgPath string, args ...string) (*cobra.Command, *bytes.Buffer, error) {
	rootCmd := newRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	fullArgs := append([]string{"--config", cfgPath}, args...)
	rootCmd.SetArgs(fullArgs)
	err := rootCmd.Execute()
	return rootCmd, buf, err
}

func assertContains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Errorf("expected %q to contain %q", got, want)
	}
}

func assertNotContains(t *testing.T, got, unwanted string) {
	t.Helper()
	if strings.Contains(got, unwanted) {
		t.Errorf("expected %q not to contain %q", got, unwanted)
	}
}

func readFileString(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
