package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jeroenjanssens/velocirepo/internal/config"
	"github.com/spf13/cobra"
)

type projectSourceField struct {
	key          string
	flagName     string
	flagUsage    string
	addPrompt    string
	updatePrompt string
	jsonKeys     []string
	csvColumns   []string
	detected     func(config.Detected) (string, string)
	validate     func(string) error
}

var projectSourceFields = []projectSourceField{
	{
		key:          "github",
		flagName:     "github",
		flagUsage:    "GitHub owner/repo",
		addPrompt:    "GitHub (owner/repo)",
		updatePrompt: "GitHub (owner/repo)",
		jsonKeys:     []string{"github"},
		csvColumns:   []string{"github"},
		detected: func(d config.Detected) (string, string) {
			return d.GitHub, d.GitHubSource
		},
		validate: validateGitHubList,
	},
	{
		key:          "github-traffic",
		flagName:     "github-traffic",
		flagUsage:    "GitHub owner/repo for traffic data",
		addPrompt:    "GitHub traffic (owner/repo)",
		updatePrompt: "GitHub traffic (owner/repo)",
		jsonKeys:     []string{"github_traffic", "github-traffic"},
		csvColumns:   []string{"github-traffic", "github_traffic"},
		detected: func(d config.Detected) (string, string) {
			return d.GitHub, d.GitHubSource
		},
		validate: validateGitHubList,
	},
	{
		key:          "pypi",
		flagName:     "pypi",
		flagUsage:    "PyPI package name",
		addPrompt:    "PyPI package",
		updatePrompt: "PyPI package",
		jsonKeys:     []string{"pypi"},
		csvColumns:   []string{"pypi"},
		detected: func(d config.Detected) (string, string) {
			return d.PyPI, d.PyPISource
		},
	},
	{
		key:          "cran",
		flagName:     "cran",
		flagUsage:    "CRAN package name",
		addPrompt:    "CRAN package",
		updatePrompt: "CRAN package",
		jsonKeys:     []string{"cran"},
		csvColumns:   []string{"cran"},
		detected: func(d config.Detected) (string, string) {
			return d.CRAN, d.CRANSource
		},
	},
	{
		key:          "homebrew",
		flagName:     "homebrew",
		flagUsage:    "Homebrew formula",
		addPrompt:    "Homebrew formula",
		updatePrompt: "Homebrew formula",
		jsonKeys:     []string{"homebrew"},
		csvColumns:   []string{"homebrew"},
	},
	{
		key:          "plausible",
		flagName:     "plausible",
		flagUsage:    "Plausible site ID",
		addPrompt:    "Plausible site ID",
		updatePrompt: "Plausible site ID",
		jsonKeys:     []string{"plausible"},
		csvColumns:   []string{"plausible"},
	},
	{
		key:          "openvsx",
		flagName:     "openvsx",
		flagUsage:    "OpenVSX extension (publisher/extension)",
		addPrompt:    "OpenVSX extension",
		updatePrompt: "OpenVSX extension",
		jsonKeys:     []string{"openvsx"},
		csvColumns:   []string{"openvsx"},
		detected: func(d config.Detected) (string, string) {
			return d.OpenVSX, d.OpenVSXSource
		},
	},
	{
		key:          "youtube",
		flagName:     "youtube",
		flagUsage:    "YouTube channel (@handle), playlist (PLxxx), or video ID",
		addPrompt:    "YouTube (@handle, PLxxx, or video ID)",
		updatePrompt: "YouTube (@handle, PLxxx, or video ID)",
		jsonKeys:     []string{"youtube"},
		csvColumns:   []string{"youtube"},
	},
	{
		key:          "linkedin",
		flagName:     "linkedin",
		flagUsage:    "LinkedIn URN",
		addPrompt:    "LinkedIn URN",
		updatePrompt: "LinkedIn URN",
		jsonKeys:     []string{"linkedin"},
		csvColumns:   []string{"linkedin"},
	},
}

type projectSourceInputs struct {
	values []string
}

func newProjectSourceInputs() *projectSourceInputs {
	return &projectSourceInputs{values: make([]string, len(projectSourceFields))}
}

func (in *projectSourceInputs) registerFlags(cmd *cobra.Command) {
	for i, field := range projectSourceFields {
		cmd.Flags().StringVar(&in.values[i], field.flagName, "", field.flagUsage)
	}
}

func (in *projectSourceInputs) anyNonEmpty() bool {
	for _, value := range in.values {
		if value != "" {
			return true
		}
	}
	return false
}

func (in *projectSourceInputs) changedUpdates(cmd *cobra.Command) (map[string]string, error) {
	updates := make(map[string]string)
	for i, field := range projectSourceFields {
		if !cmd.Flags().Changed(field.flagName) {
			continue
		}
		value := in.values[i]
		if err := validateProjectSourceValue(field, value); err != nil {
			return nil, err
		}
		updates[field.key] = value
	}
	return updates, nil
}

func (in *projectSourceInputs) toProject(name string) (config.Project, error) {
	p := config.Project{Name: name}
	for i, field := range projectSourceFields {
		value := in.values[i]
		if err := validateProjectSourceValue(field, value); err != nil {
			return config.Project{}, err
		}
		p.SetSourceValues(field.key, toStringList(value))
	}
	return p, nil
}

func projectFromRawSourceValues(name string, values map[string]string) (config.Project, error) {
	p := config.Project{Name: name}
	for _, field := range projectSourceFields {
		value := values[field.key]
		if err := validateProjectSourceValue(field, value); err != nil {
			return config.Project{}, err
		}
		p.SetSourceValues(field.key, toStringList(value))
	}
	return p, nil
}

func validateProjectSourceValue(field projectSourceField, value string) error {
	if value == "" || field.validate == nil {
		return nil
	}
	return field.validate(value)
}

func validateGitHubList(value string) error {
	for _, repo := range parseCommaSeparated(value) {
		if !validGitHubRe.MatchString(repo) {
			return fmt.Errorf("invalid GitHub repo %q: must be owner/repo", repo)
		}
	}
	return nil
}

func sourceValueFromCSV(row []string, colIndex map[string]int, field projectSourceField) string {
	for _, col := range field.csvColumns {
		idx, ok := colIndex[col]
		if ok && idx < len(row) {
			value := row[idx]
			if value != "" {
				return value
			}
		}
	}
	return ""
}

func applyJSONSourceValues(project *config.Project, item map[string]json.RawMessage) error {
	for _, field := range projectSourceFields {
		values, err := jsonSourceList(item, field.jsonKeys)
		if err != nil {
			return err
		}
		project.SetSourceValues(field.key, values)
	}
	return nil
}

func jsonSourceList(item map[string]json.RawMessage, keys []string) (config.StringList, error) {
	for _, key := range keys {
		raw, ok := item[key]
		if !ok {
			continue
		}
		var single string
		if err := json.Unmarshal(raw, &single); err == nil {
			values := toStringList(single)
			if len(values) > 0 {
				return values, nil
			}
			continue
		}
		var many []string
		if err := json.Unmarshal(raw, &many); err != nil {
			return nil, fmt.Errorf("parse %q: expected string or string array", key)
		}
		values := cleanStringList(many)
		if len(values) > 0 {
			return values, nil
		}
	}
	return nil, nil
}

func cleanStringList(values []string) config.StringList {
	out := make(config.StringList, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func jsonString(item map[string]json.RawMessage, key string) (string, error) {
	raw, ok := item[key]
	if !ok {
		return "", nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", fmt.Errorf("parse %q: expected string", key)
	}
	return value, nil
}
