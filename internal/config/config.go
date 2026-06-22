package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// StringList is a TOML type that accepts either a single string or an array of strings.
type StringList []string

func (s *StringList) UnmarshalTOML(data interface{}) error {
	switch v := data.(type) {
	case string:
		*s = StringList{v}
	case []interface{}:
		for _, item := range v {
			str, ok := item.(string)
			if !ok {
				return fmt.Errorf("expected string in array, got %T", item)
			}
			*s = append(*s, str)
		}
	default:
		return fmt.Errorf("expected string or array, got %T", data)
	}
	return nil
}

func (s StringList) IsEmpty() bool {
	return len(s) == 0
}

func (s StringList) First() string {
	if len(s) == 0 {
		return ""
	}
	return s[0]
}

func (s StringList) String() string {
	if len(s) == 0 {
		return ""
	}
	if len(s) == 1 {
		return s[0]
	}
	result := s[0]
	for _, v := range s[1:] {
		result += ", " + v
	}
	return result
}

type Project struct {
	Name        string     `toml:"name"`
	Description string     `toml:"description"`
	Color       string     `toml:"color"`
	Tags        StringList `toml:"tags"`
	Website     string     `toml:"website"`
	Logo        string     `toml:"logo"`

	GitHubEvents  StringList `toml:"github-events"`
	GitHubTraffic StringList `toml:"github-traffic"`
	PyPI          StringList `toml:"pypi"`
	CRAN          StringList `toml:"cran"`
	Homebrew      StringList `toml:"homebrew"`
	Plausible     StringList `toml:"plausible"`
	OpenVSX       StringList `toml:"openvsx"`
	YouTube       StringList `toml:"youtube"`
}

type DataConfig struct {
	Dir string `toml:"dir"`
}

type SettingsConfig struct {
	EndDate string `toml:"end_date"`
}

type ViewsConfig struct {
	Dir    string     `toml:"dir"`
	Source string     `toml:"source"`
	Items  []ViewItem `toml:"items"`
}

type ViewItem struct {
	Path   string `toml:"path"`
	Output string `toml:"output"`
	Venv   string `toml:"venv"`
	Source string `toml:"source"`
}

type Config struct {
	Data     DataConfig         `toml:"data"`
	Settings SettingsConfig     `toml:"settings"`
	Views    ViewsConfig        `toml:"views"`
	Projects map[string]Project `toml:"projects"`

	// Computed fields
	Dir string `toml:"-"`
}

func (c *Config) ViewsDir() string {
	dir := c.Views.Dir
	if dir == "" {
		dir = "velocirepo/views"
	}
	if filepath.IsAbs(dir) {
		return dir
	}
	return filepath.Join(c.Dir, dir)
}

func (c *Config) ViewsSource() string {
	if c.Views.Source != "" {
		return c.Views.Source
	}
	return "parquet"
}

func (c *Config) DataDir() string {
	dir := c.Data.Dir
	if dir == "" {
		dir = "velocirepo/data"
	}
	if filepath.IsAbs(dir) {
		return dir
	}
	return filepath.Join(c.Dir, dir)
}

func (c *Config) ResolveProjects() map[string]Project {
	if len(c.Projects) > 0 {
		return c.Projects
	}
	return nil
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
