package config

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type ValidationResult struct {
	Source     string
	Value      string
	OK         bool
	Error      string
	StatusCode int
}

type ValidationOptions struct {
	Client  *http.Client
	Timeout time.Duration
	Token   string
}

func ValidateSource(ctx context.Context, opts ValidationOptions, sourceType string, value string) ValidationResult {
	result := ValidationResult{Source: sourceType, Value: value}

	var url string
	switch sourceType {
	case "github":
		url = "https://api.github.com/repos/" + value
	case "pypi":
		url = "https://pypi.org/pypi/" + value + "/json"
	case "cran":
		url = "https://cranlogs.r-pkg.org/downloads/total/last-day/" + value
	case "homebrew":
		url = "https://formulae.brew.sh/api/formula/" + value + ".json"
	case "openvsx":
		url = "https://open-vsx.org/api/" + value
	default:
		result.Error = "unknown source type"
		return result
	}

	client := opts.Client
	if client == nil {
		client = &http.Client{Timeout: opts.Timeout}
	}

	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	if sourceType == "github" && opts.Token != "" {
		req.Header.Set("Authorization", "Bearer "+opts.Token)
	}

	resp, err := client.Do(req)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	resp.Body.Close()

	result.StatusCode = resp.StatusCode
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		result.OK = true
	} else {
		result.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	return result
}

func ValidateProject(ctx context.Context, opts ValidationOptions, id string, project Project) []ValidationResult {
	var results []ValidationResult

	type sourceEntry struct {
		name  string
		value string
	}

	var entries []sourceEntry
	if project.GitHub != "" {
		entries = append(entries, sourceEntry{"github", project.GitHub})
	}
	if project.PyPI != "" {
		entries = append(entries, sourceEntry{"pypi", project.PyPI})
	}
	if project.CRAN != "" {
		entries = append(entries, sourceEntry{"cran", project.CRAN})
	}
	if project.Homebrew != "" {
		entries = append(entries, sourceEntry{"homebrew", project.Homebrew})
	}
	if project.OpenVSX != "" {
		entries = append(entries, sourceEntry{"openvsx", project.OpenVSX})
	}

	for _, e := range entries {
		results = append(results, ValidateSource(ctx, opts, e.name, e.value))
	}

	return results
}
