package views

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func Render(view View) error {
	script := filepath.Join(view.Dir, "render.sh")
	cmd := exec.Command("bash", script)
	cmd.Dir = view.Dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("render %s: %w", view.Name, err)
	}
	return nil
}

func ServeCmd(view View) (*exec.Cmd, error) {
	script := filepath.Join(view.Dir, "serve.sh")
	if _, err := os.Stat(script); err != nil {
		return nil, fmt.Errorf("no serve.sh in %s; create one to enable live serving", view.Dir)
	}
	cmd := exec.Command("bash", script)
	cmd.Dir = view.Dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd, nil
}
