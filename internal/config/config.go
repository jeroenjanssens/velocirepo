package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Project struct {
	Name      string `toml:"name"`
	GitHub    string `toml:"github"`
	PyPI      string `toml:"pypi"`
	CRAN      string `toml:"cran"`
	Plausible string `toml:"plausible"`
	OpenVSX   string `toml:"openvsx"`
}

type DataConfig struct {
	Dir string `toml:"dir"`
}

type SettingsConfig struct {
	EndDate string `toml:"end_date"`
}

type Config struct {
	Data     DataConfig        `toml:"data"`
	Settings SettingsConfig    `toml:"settings"`
	Project  *Project          `toml:"project"`
	Projects map[string]Project `toml:"projects"`

	// Computed fields
	Dir string `toml:"-"`
}

func (c *Config) DataDir() string {
	dir := c.Data.Dir
	if dir == "" {
		dir = "data"
	}
	if filepath.IsAbs(dir) {
		return dir
	}
	return filepath.Join(c.Dir, dir)
}

func (c *Config) ResolveProjects() map[string]Project {
	if c.Projects != nil && len(c.Projects) > 0 {
		return c.Projects
	}
	if c.Project != nil {
		id := projectID(c.Project)
		return map[string]Project{id: *c.Project}
	}
	return nil
}

func projectID(p *Project) string {
	if p.GitHub != "" {
		_, repo := splitRepo(p.GitHub)
		return repo
	}
	if p.PyPI != "" {
		return p.PyPI
	}
	if p.CRAN != "" {
		return p.CRAN
	}
	return "project"
}

func splitRepo(repo string) (string, string) {
	for i, c := range repo {
		if c == '/' {
			return repo[:i], repo[i+1:]
		}
	}
	return repo, repo
}

func Load(path string) (*Config, error) {
	if path == "" {
		path = os.Getenv("VELOCIREPO_CONFIG")
	}
	if path == "" {
		var err error
		path, err = discover()
		if err != nil {
			return nil, err
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	cfg.Dir = filepath.Dir(path)
	return &cfg, nil
}

func discover() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	for {
		path := filepath.Join(dir, "velocirepo.toml")
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("velocirepo.toml not found (searched from working directory to root)")
		}
		dir = parent
	}
}
