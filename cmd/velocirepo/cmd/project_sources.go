package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jeroenjanssens/velocirepo/internal/config"
	"github.com/jeroenjanssens/velocirepo/internal/sourceinfo"
	"github.com/spf13/cobra"
)

type projectSourceField struct {
	sourceinfo.Descriptor
	detected func(config.Detected) (string, string)
	validate func(string) error
}

var projectSourceFields = buildProjectSourceFields()

func buildProjectSourceFields() []projectSourceField {
	detectedBySource := map[string]func(config.Detected) (string, string){
		"github": func(d config.Detected) (string, string) {
			return d.GitHub, d.GitHubSource
		},
		"github-traffic": func(d config.Detected) (string, string) {
			return d.GitHub, d.GitHubSource
		},
		"pypi": func(d config.Detected) (string, string) {
			return d.PyPI, d.PyPISource
		},
		"cran": func(d config.Detected) (string, string) {
			return d.CRAN, d.CRANSource
		},
		"openvsx": func(d config.Detected) (string, string) {
			return d.OpenVSX, d.OpenVSXSource
		},
	}
	validators := map[string]func(string) error{
		"github":         validateGitHubList,
		"github-traffic": validateGitHubList,
	}

	descriptors := sourceinfo.All()
	fields := make([]projectSourceField, 0, len(descriptors))
	for _, desc := range descriptors {
		fields = append(fields, projectSourceField{
			Descriptor: desc,
			detected:   detectedBySource[desc.Name],
			validate:   validators[desc.Name],
		})
	}
	return fields
}

type projectSourceInputs struct {
	values []string
}

func newProjectSourceInputs() *projectSourceInputs {
	return &projectSourceInputs{values: make([]string, len(projectSourceFields))}
}

func (in *projectSourceInputs) registerFlags(cmd *cobra.Command) {
	for i, field := range projectSourceFields {
		cmd.Flags().StringVar(&in.values[i], field.CLIFlag, "", field.CLIUsage)
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
		if !cmd.Flags().Changed(field.CLIFlag) {
			continue
		}
		value := in.values[i]
		if err := validateProjectSourceValue(field, value); err != nil {
			return nil, err
		}
		updates[field.TOMLKey] = value
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
		p.SetSourceValues(field.Name, toStringList(value))
	}
	return p, nil
}

func projectFromRawSourceValues(name string, values map[string]string) (config.Project, error) {
	p := config.Project{Name: name}
	for _, field := range projectSourceFields {
		value := values[field.Name]
		if err := validateProjectSourceValue(field, value); err != nil {
			return config.Project{}, err
		}
		p.SetSourceValues(field.Name, toStringList(value))
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
	for _, col := range field.CSVColumns {
		idx, ok := colIndex[col]
		if ok && idx < len(row) {
			value := strings.TrimSpace(row[idx])
			if value != "" {
				return value
			}
		}
	}
	return ""
}

func applyJSONSourceValues(project *config.Project, item map[string]json.RawMessage) error {
	for _, field := range projectSourceFields {
		values, err := jsonSourceList(item, field.JSONKeys)
		if err != nil {
			return err
		}
		project.SetSourceValues(field.Name, values)
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
