package views

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type RendererInfo struct {
	Binary     string
	VersionCmd []string
	InstallURL string
}

var rendererChecks = map[Framework]RendererInfo{
	FrameworkQuarto:  {"quarto", []string{"quarto", "--version"}, "https://quarto.org/docs/get-started/"},
	FrameworkJupyter: {"jupyter", []string{"jupyter", "--version"}, "https://jupyter.org/install"},
	FrameworkMarimo:  {"python", []string{"python", "--version"}, "https://www.python.org/downloads/"},
	FrameworkR:       {"Rscript", []string{"Rscript", "--version"}, "https://cran.r-project.org/"},
	FrameworkSQL:     {"ggsql", []string{"ggsql", "--version"}, "https://github.com/jeroenjanssens/ggsql"},
}

var serveRendererChecks = map[Framework]RendererInfo{
	FrameworkMarimo: {"marimo", []string{"marimo", "--version"}, "https://docs.marimo.io/getting_started/"},
}

func CheckRenderer(fw Framework, venv string) (string, error) {
	info, ok := rendererChecks[fw]
	if !ok {
		return "", fmt.Errorf("no renderer defined for framework %q", fw)
	}

	binary := info.Binary
	if venv != "" {
		venvBin := filepath.Join(venv, "bin", binary)
		if _, err := os.Stat(venvBin); err == nil {
			binary = venvBin
		}
	}

	path, err := exec.LookPath(binary)
	if err != nil {
		return "", fmt.Errorf("%s not found; install from %s", info.Binary, info.InstallURL)
	}

	cmd := exec.Command(info.VersionCmd[0], info.VersionCmd[1:]...)
	if venv != "" {
		cmd.Env = prependVenvPath(venv)
	}
	out, err := cmd.Output()
	if err != nil {
		return path, nil
	}
	ver := strings.TrimSpace(string(out))
	if lines := strings.Split(ver, "\n"); len(lines) > 0 {
		ver = lines[0]
	}
	return ver, nil
}

func Render(view View) error {
	if err := os.MkdirAll(filepath.Dir(view.Output), 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	var cmd *exec.Cmd
	switch view.Framework {
	case FrameworkQuarto:
		outputDir := filepath.Dir(view.Output)
		cmd = exec.Command("quarto", "render", view.Path, "--output-dir", outputDir)
	case FrameworkJupyter:
		cmd = exec.Command("jupyter", "nbconvert", "--execute", "--to", "html", view.Path, "--output", view.Output)
	case FrameworkMarimo:
		cmd = exec.Command("python", view.Path)
	case FrameworkR:
		cmd = exec.Command("Rscript", view.Path)
	case FrameworkSQL:
		cmd = exec.Command("ggsql", "run", view.Path, "--output", view.Output)
	default:
		return fmt.Errorf("unsupported framework %q", view.Framework)
	}

	if view.Venv != "" {
		cmd.Env = prependVenvPath(view.Venv)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("render %s: %w", view.Name, err)
	}
	return nil
}

func ServeCmd(view View, port string) (*exec.Cmd, error) {
	var cmd *exec.Cmd
	switch view.Framework {
	case FrameworkQuarto:
		args := []string{"preview", view.Path}
		if port != "" {
			args = append(args, "--port", port)
		}
		cmd = exec.Command("quarto", args...)
	case FrameworkMarimo:
		if info, ok := serveRendererChecks[FrameworkMarimo]; ok {
			if _, err := exec.LookPath(info.Binary); err != nil {
				return nil, fmt.Errorf("%s not found; install from %s", info.Binary, info.InstallURL)
			}
		}
		args := []string{"edit", view.Path}
		if port != "" {
			args = append(args, "--port", port)
		}
		cmd = exec.Command("marimo", args...)
	case FrameworkJupyter:
		args := []string{"notebook", view.Path}
		if port != "" {
			args = append(args, "--port", port)
		}
		cmd = exec.Command("jupyter", args...)
	case FrameworkR, FrameworkSQL:
		return nil, fmt.Errorf("framework %q does not support live serving; use render-view instead", view.Framework)
	default:
		return nil, fmt.Errorf("unsupported framework %q", view.Framework)
	}

	if view.Venv != "" {
		cmd.Env = prependVenvPath(view.Venv)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd, nil
}

func prependVenvPath(venv string) []string {
	env := os.Environ()
	venvBin := filepath.Join(venv, "bin")
	for i, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			env[i] = "PATH=" + venvBin + string(os.PathListSeparator) + e[5:]
			return env
		}
	}
	return append(env, "PATH="+venvBin)
}
