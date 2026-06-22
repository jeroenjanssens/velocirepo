package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/jeroenjanssens/velocirepo/internal/badge"
	"github.com/jeroenjanssens/velocirepo/internal/config"
	"github.com/jeroenjanssens/velocirepo/internal/fetch"
	"github.com/jeroenjanssens/velocirepo/internal/store"
	"github.com/jeroenjanssens/velocirepo/internal/version"
	"github.com/jeroenjanssens/velocirepo/internal/views"
	"github.com/mark3labs/mcp-go/mcp"
)

type handlers struct {
	cfg *config.Config
}

func (h *handlers) projectInfos() []store.ProjectInfo {
	projects := h.cfg.ResolveProjects()
	if projects == nil {
		return nil
	}
	infos := make([]store.ProjectInfo, 0, len(projects))
	for id, p := range projects {
		infos = append(infos, store.ProjectInfo{
			ID:          id,
			Name:        p.Name,
			Description: p.Description,
			Color:       p.Color,
			Tags:        []string(p.Tags),
			Website:     p.Website,
			Logo:        p.Logo,
		})
	}
	return infos
}

func (h *handlers) cfgFilePath() string {
	if h.cfg != nil && h.cfg.Dir != "" {
		return filepath.Join(h.cfg.Dir, "velocirepo.toml")
	}
	return "velocirepo.toml"
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(text)},
	}
}

func jsonResult(v any) *mcp.CallToolResult {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return errorResult(fmt.Sprintf("marshal result: %v", err))
	}
	return textResult(string(data))
}

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(msg)},
		IsError: true,
	}
}

func (h *handlers) handleQuery(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sql, err := req.RequireString("sql")
	if err != nil {
		return errorResult(err.Error()), nil
	}

	limit := req.GetInt("limit", 1000)
	if limit > 0 && !strings.Contains(strings.ToUpper(sql), "LIMIT") {
		sql = fmt.Sprintf("SELECT * FROM (%s) LIMIT %d", sql, limit)
	}

	results, cols, err := store.QueryLive(h.cfg.DataDir(), h.projectInfos(), sql)
	if err != nil {
		return errorResult(fmt.Sprintf("query error: %v", err)), nil
	}

	output := struct {
		Columns  []string                 `json:"columns"`
		Rows     []map[string]interface{} `json:"rows"`
		RowCount int                      `json:"row_count"`
	}{
		Columns:  cols,
		Rows:     results,
		RowCount: len(results),
	}

	return jsonResult(output), nil
}

func (h *handlers) handleSchema(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cols, err := store.SchemaLive(h.cfg.DataDir(), h.projectInfos())
	if err != nil {
		return errorResult(fmt.Sprintf("schema error: %v", err)), nil
	}

	type column struct {
		Table  string `json:"table"`
		Column string `json:"column"`
		Type   string `json:"type"`
	}

	result := make([]column, len(cols))
	for i, c := range cols {
		result[i] = column{Table: c.Table, Column: c.Column, Type: c.Type}
	}

	return jsonResult(result), nil
}

func (h *handlers) handleListProjects(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projects := h.cfg.ResolveProjects()

	type projectEntry struct {
		ID            string   `json:"id"`
		Name          string   `json:"name"`
		GitHubEvents  []string `json:"github_events,omitempty"`
		GitHubTraffic []string `json:"github_traffic,omitempty"`
		PyPI          []string `json:"pypi,omitempty"`
		CRAN          []string `json:"cran,omitempty"`
		Homebrew      []string `json:"homebrew,omitempty"`
		Plausible     []string `json:"plausible,omitempty"`
		OpenVSX       []string `json:"openvsx,omitempty"`
		YouTube       []string `json:"youtube,omitempty"`
	}

	var list []projectEntry
	for id, p := range projects {
		list = append(list, projectEntry{
			ID:            id,
			Name:          p.Name,
			GitHubEvents:  []string(p.GitHubEvents),
			GitHubTraffic: []string(p.GitHubTraffic),
			PyPI:          []string(p.PyPI),
			CRAN:          []string(p.CRAN),
			Homebrew:      []string(p.Homebrew),
			Plausible:     []string(p.Plausible),
			OpenVSX:       []string(p.OpenVSX),
			YouTube:       []string(p.YouTube),
		})
	}

	return jsonResult(list), nil
}

func (h *handlers) handleShowProject(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return errorResult(err.Error()), nil
	}

	projects := h.cfg.ResolveProjects()
	proj, exists := projects[id]
	if !exists {
		return errorResult(fmt.Sprintf("project %q not found in config", id)), nil
	}

	dataDir := h.cfg.DataDir()

	type sourceStats struct {
		Source   string `json:"source"`
		LastDate string `json:"last_date,omitempty"`
		Records  int    `json:"records"`
	}

	var sources []string
	if !proj.GitHubEvents.IsEmpty() {
		sources = append(sources, "github")
	}
	if !proj.GitHubTraffic.IsEmpty() {
		sources = append(sources, "github-traffic")
	}
	if !proj.PyPI.IsEmpty() {
		sources = append(sources, "pypi")
	}
	if !proj.CRAN.IsEmpty() {
		sources = append(sources, "cran")
	}
	if !proj.Homebrew.IsEmpty() {
		sources = append(sources, "homebrew")
	}
	if !proj.Plausible.IsEmpty() {
		sources = append(sources, "plausible")
	}
	if !proj.OpenVSX.IsEmpty() {
		sources = append(sources, "openvsx")
	}
	if !proj.YouTube.IsEmpty() {
		sources = append(sources, "youtube")
	}

	var stats []sourceStats
	for _, src := range sources {
		srcDir := src
		if src == "github" {
			srcDir = "github"
		}
		dir := filepath.Join(dataDir, srcDir, id)
		lastDate, records := scanDir(dir)
		stats = append(stats, sourceStats{
			Source:   src,
			LastDate: lastDate,
			Records:  records,
		})
	}

	output := struct {
		ID            string        `json:"id"`
		Name          string        `json:"name"`
		GitHubEvents  []string      `json:"github_events,omitempty"`
		GitHubTraffic []string      `json:"github_traffic,omitempty"`
		PyPI          []string      `json:"pypi,omitempty"`
		CRAN          []string      `json:"cran,omitempty"`
		Homebrew      []string      `json:"homebrew,omitempty"`
		Plausible     []string      `json:"plausible,omitempty"`
		OpenVSX       []string      `json:"openvsx,omitempty"`
		YouTube       []string      `json:"youtube,omitempty"`
		Sources       []sourceStats `json:"sources"`
	}{
		ID:            id,
		Name:          proj.Name,
		GitHubEvents:  []string(proj.GitHubEvents),
		GitHubTraffic: []string(proj.GitHubTraffic),
		PyPI:          []string(proj.PyPI),
		CRAN:          []string(proj.CRAN),
		Homebrew:      []string(proj.Homebrew),
		Plausible:     []string(proj.Plausible),
		OpenVSX:       []string(proj.OpenVSX),
		YouTube:       []string(proj.YouTube),
		Sources:       stats,
	}

	return jsonResult(output), nil
}

func scanDir(dir string) (string, int) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", 0
	}

	var lastDate string
	var records int
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		recs, _ := store.ReadRecords(path)
		records += len(recs)

		datePart := strings.TrimSuffix(e.Name(), ".jsonl")
		if datePart > lastDate {
			lastDate = datePart
		}
	}
	return lastDate, records
}

var badgePresets = map[string]struct {
	label string
	query string
	color string
}{
	"stars":     {"stars", "SELECT COUNT(*) AS value FROM github_events WHERE event_type = 'star'", "#007ec6"},
	"forks":     {"forks", "SELECT COUNT(*) AS value FROM github_events WHERE event_type = 'fork'", "#007ec6"},
	"downloads": {"downloads", "SELECT MAX(value) AS value FROM metrics WHERE metric = 'downloads' OR metric = 'total_downloads'", "#44cc11"},
	"pageviews": {"pageviews", "SELECT SUM(value) AS value FROM metrics WHERE metric = 'pageviews'", "#44cc11"},
}

func (h *handlers) handleBadge(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	badgeType, err := req.RequireString("type")
	if err != nil {
		return errorResult(err.Error()), nil
	}

	project := req.GetString("project", "")
	query := req.GetString("query", "")
	label := req.GetString("label", "")
	style := req.GetString("style", "flat")
	color := req.GetString("color", "")

	var q string
	if badgeType == "custom" {
		if query == "" {
			return errorResult("--query is required for custom badges"), nil
		}
		if label == "" {
			return errorResult("--label is required for custom badges"), nil
		}
		q = query
		if project != "" {
			q = fmt.Sprintf("SELECT * FROM (%s) WHERE project = '%s'", q, project)
		}
		if color == "" {
			color = "#007ec6"
		}
	} else {
		preset, ok := badgePresets[badgeType]
		if !ok {
			return errorResult(fmt.Sprintf("unknown badge type %q (available: stars, forks, downloads, pageviews, custom)", badgeType)), nil
		}
		q = preset.query
		if project != "" {
			q += fmt.Sprintf(" AND project = '%s'", project)
		}
		if label == "" {
			label = preset.label
		}
		if color == "" {
			color = preset.color
		}
	}

	results, _, err := store.QueryLive(h.cfg.DataDir(), h.projectInfos(), q)
	if err != nil {
		return errorResult(fmt.Sprintf("badge query: %v", err)), nil
	}

	msg := "0"
	if len(results) > 0 {
		for _, v := range results[0] {
			switch val := v.(type) {
			case int64:
				msg = badge.FormatNumber(val)
			case float64:
				msg = badge.FormatNumber(int64(val))
			case string:
				msg = val
			default:
				msg = fmt.Sprintf("%v", v)
			}
			break
		}
	}

	svg := badge.Render(badge.Options{
		Label:      label,
		Message:    msg,
		Color:      color,
		LabelColor: "#555",
		Style:      badge.Style(style),
	})

	return textResult(svg), nil
}

func (h *handlers) handleVersion(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	output := struct {
		Version string `json:"version"`
		Commit  string `json:"commit"`
		Date    string `json:"date"`
	}{
		Version: version.Version,
		Commit:  version.Commit,
		Date:    version.Date,
	}
	return jsonResult(output), nil
}

func (h *handlers) fetchOpts(req mcp.CallToolRequest) fetch.Options {
	return fetch.Options{
		Project:   req.GetString("project", ""),
		StartDate: req.GetString("start_date", ""),
		EndDate:   req.GetString("end_date", ""),
	}
}

func (h *handlers) handleFetch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	results, err := fetch.All(ctx, h.cfg, fetch.TokensFromEnv(), h.fetchOpts(req))
	if err != nil {
		return errorResult(err.Error()), nil
	}
	return jsonResult(results), nil
}

func (h *handlers) handleFetchGitHub(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	results, err := fetch.GitHub(ctx, h.cfg, fetch.TokensFromEnv(), h.fetchOpts(req))
	if err != nil {
		return errorResult(err.Error()), nil
	}
	return jsonResult(results), nil
}

func (h *handlers) handleFetchTraffic(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	results, err := fetch.Traffic(ctx, h.cfg, fetch.TokensFromEnv(), h.fetchOpts(req))
	if err != nil {
		return errorResult(err.Error()), nil
	}
	return jsonResult(results), nil
}

func (h *handlers) handleFetchPyPI(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	results, err := fetch.PyPI(ctx, h.cfg, fetch.TokensFromEnv(), h.fetchOpts(req))
	if err != nil {
		return errorResult(err.Error()), nil
	}
	return jsonResult(results), nil
}

func (h *handlers) handleFetchCRAN(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	results, err := fetch.CRAN(ctx, h.cfg, fetch.TokensFromEnv(), h.fetchOpts(req))
	if err != nil {
		return errorResult(err.Error()), nil
	}
	return jsonResult(results), nil
}

func (h *handlers) handleFetchHomebrew(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	results, err := fetch.Homebrew(ctx, h.cfg, fetch.TokensFromEnv(), h.fetchOpts(req))
	if err != nil {
		return errorResult(err.Error()), nil
	}
	return jsonResult(results), nil
}

func (h *handlers) handleFetchPlausible(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	results, err := fetch.Plausible(ctx, h.cfg, fetch.TokensFromEnv(), h.fetchOpts(req))
	if err != nil {
		return errorResult(err.Error()), nil
	}
	return jsonResult(results), nil
}

func (h *handlers) handleFetchOpenVSX(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	results, err := fetch.OpenVSX(ctx, h.cfg, fetch.TokensFromEnv(), h.fetchOpts(req))
	if err != nil {
		return errorResult(err.Error()), nil
	}
	return jsonResult(results), nil
}

func (h *handlers) handleFetchYouTube(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	results, err := fetch.YouTube(ctx, h.cfg, fetch.TokensFromEnv(), h.fetchOpts(req))
	if err != nil {
		return errorResult(err.Error()), nil
	}
	return jsonResult(results), nil
}

var validIDRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func (h *handlers) handleAddProject(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return errorResult(err.Error()), nil
	}

	if !validIDRe.MatchString(id) {
		return errorResult(fmt.Sprintf("invalid project ID %q: must be lowercase alphanumeric with hyphens", id)), nil
	}

	projects := h.cfg.ResolveProjects()
	if _, exists := projects[id]; exists {
		return errorResult(fmt.Sprintf("project %q already exists", id)), nil
	}

	name := req.GetString("name", id)

	proj := config.Project{
		Name:          name,
		GitHubEvents:  toStringList(req.GetString("github_events", "")),
		GitHubTraffic: toStringList(req.GetString("github_traffic", "")),
		PyPI:          toStringList(req.GetString("pypi", "")),
		CRAN:          toStringList(req.GetString("cran", "")),
		Homebrew:      toStringList(req.GetString("homebrew", "")),
		Plausible:     toStringList(req.GetString("plausible", "")),
		OpenVSX:       toStringList(req.GetString("openvsx", "")),
		YouTube:       toStringList(req.GetString("youtube", "")),
	}

	if err := config.AppendProject(h.cfgFilePath(), id, proj); err != nil {
		return errorResult(fmt.Sprintf("add project: %v", err)), nil
	}

	h.cfg.Projects[id] = proj

	return textResult(fmt.Sprintf("Added project '%s'", id)), nil
}

func (h *handlers) handleUpdateProject(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return errorResult(err.Error()), nil
	}

	projects := h.cfg.ResolveProjects()
	if _, exists := projects[id]; !exists {
		return errorResult(fmt.Sprintf("project %q not found in config", id)), nil
	}

	args := req.GetArguments()
	updates := make(map[string]string)
	var unsets []string

	fieldMap := map[string]string{
		"name":           "name",
		"github_events":  "github-events",
		"github_traffic": "github-traffic",
		"pypi":           "pypi",
		"cran":           "cran",
		"homebrew":       "homebrew",
		"plausible":      "plausible",
		"openvsx":        "openvsx",
		"youtube":        "youtube",
	}

	for argKey, tomlKey := range fieldMap {
		if val, ok := args[argKey]; ok {
			str, _ := val.(string)
			if str == "" {
				unsets = append(unsets, tomlKey)
			} else {
				updates[tomlKey] = str
			}
		}
	}

	if len(updates) == 0 && len(unsets) == 0 {
		return errorResult("no changes specified"), nil
	}

	if err := config.UpdateProject(h.cfgFilePath(), id, updates, unsets); err != nil {
		return errorResult(fmt.Sprintf("update project: %v", err)), nil
	}

	return textResult(fmt.Sprintf("Updated project '%s'", id)), nil
}

func (h *handlers) handleRemoveProject(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return errorResult(err.Error()), nil
	}

	projects := h.cfg.ResolveProjects()
	if _, exists := projects[id]; !exists {
		return errorResult(fmt.Sprintf("project %q not found in config", id)), nil
	}

	if err := config.RemoveProject(h.cfgFilePath(), id); err != nil {
		return errorResult(fmt.Sprintf("remove project: %v", err)), nil
	}

	deleteData := false
	if args := req.GetArguments(); args != nil {
		if v, ok := args["delete_data"]; ok {
			deleteData, _ = v.(bool)
		}
	}

	if deleteData {
		dataDir := h.cfg.DataDir()
		for _, src := range []string{"github", "github-traffic", "pypi", "cran", "homebrew", "plausible", "openvsx", "youtube"} {
			dir := filepath.Join(dataDir, src, id)
			if _, err := os.Stat(dir); err == nil {
				os.RemoveAll(dir)
			}
		}
	}

	delete(h.cfg.Projects, id)

	return textResult(fmt.Sprintf("Removed project '%s'", id)), nil
}

func (h *handlers) handleRenameProject(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	oldID, err := req.RequireString("old_id")
	if err != nil {
		return errorResult(err.Error()), nil
	}
	newID, err := req.RequireString("new_id")
	if err != nil {
		return errorResult(err.Error()), nil
	}

	if !validIDRe.MatchString(newID) {
		return errorResult(fmt.Sprintf("invalid project ID %q: must be lowercase alphanumeric with hyphens", newID)), nil
	}

	projects := h.cfg.ResolveProjects()
	if _, exists := projects[oldID]; !exists {
		return errorResult(fmt.Sprintf("project %q not found in config", oldID)), nil
	}
	if _, exists := projects[newID]; exists {
		return errorResult(fmt.Sprintf("project %q already exists", newID)), nil
	}

	if err := config.RenameSection(h.cfgFilePath(), oldID, newID); err != nil {
		return errorResult(fmt.Sprintf("rename project: %v", err)), nil
	}

	dataDir := h.cfg.DataDir()
	for _, src := range []string{"github", "github-traffic", "pypi", "cran", "homebrew", "plausible", "openvsx", "youtube"} {
		oldDir := filepath.Join(dataDir, src, oldID)
		newDir := filepath.Join(dataDir, src, newID)
		if _, err := os.Stat(oldDir); err == nil {
			os.Rename(oldDir, newDir)
		}
	}

	if proj, ok := h.cfg.Projects[oldID]; ok {
		h.cfg.Projects[newID] = proj
		delete(h.cfg.Projects, oldID)
	}

	return textResult(fmt.Sprintf("Renamed project '%s' → '%s'", oldID, newID)), nil
}

func (h *handlers) handleImportProjects(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	githubOrg := req.GetString("github_org", "")
	githubUser := req.GetString("github_user", "")
	filter := req.GetString("filter", "")
	skipExisting := false
	if args := req.GetArguments(); args != nil {
		if v, ok := args["skip_existing"]; ok {
			skipExisting, _ = v.(bool)
		}
	}

	sources := 0
	if githubOrg != "" {
		sources++
	}
	if githubUser != "" {
		sources++
	}
	if sources == 0 {
		return errorResult("specify github_org or github_user"), nil
	}
	if sources > 1 {
		return errorResult("specify only one of github_org or github_user"), nil
	}

	var endpoint string
	if githubOrg != "" {
		endpoint = "orgs/" + githubOrg + "/repos"
	} else {
		endpoint = "users/" + githubUser + "/repos"
	}

	entries, err := fetchGitHubRepos(ctx, endpoint, filter)
	if err != nil {
		return errorResult(fmt.Sprintf("import: %v", err)), nil
	}

	existing := h.cfg.ResolveProjects()
	var added []string
	cfgPath := h.cfgFilePath()

	for _, entry := range entries {
		if _, exists := existing[entry.ID]; exists {
			if skipExisting {
				continue
			}
			return errorResult(fmt.Sprintf("project %q already exists (use skip_existing)", entry.ID)), nil
		}
		if err := config.AppendProject(cfgPath, entry.ID, entry.Project); err != nil {
			return errorResult(fmt.Sprintf("add %s: %v", entry.ID, err)), nil
		}
		h.cfg.Projects[entry.ID] = entry.Project
		added = append(added, entry.ID)
	}

	return textResult(fmt.Sprintf("Imported %d projects: %s", len(added), strings.Join(added, ", "))), nil
}

type importEntry struct {
	ID      string
	Project config.Project
}

func fetchGitHubRepos(ctx context.Context, endpoint string, filter string) ([]importEntry, error) {
	token := os.Getenv("GITHUB_TOKEN")
	client := &http.Client{Timeout: 30 * time.Second}

	var allRepos []importEntry
	page := 1

	for {
		url := fmt.Sprintf("https://api.github.com/%s?type=public&per_page=100&page=%d", endpoint, page)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		req.Header.Set("Accept", "application/vnd.github+json")

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("GitHub API request: %w", err)
		}

		if resp.StatusCode != 200 {
			resp.Body.Close()
			return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
		}

		var repos []struct {
			Name     string `json:"name"`
			FullName string `json:"full_name"`
			Fork     bool   `json:"fork"`
			Archived bool   `json:"archived"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("parse GitHub response: %w", err)
		}
		resp.Body.Close()

		if len(repos) == 0 {
			break
		}

		for _, r := range repos {
			if r.Fork || r.Archived {
				continue
			}
			if filter != "" {
				matched, _ := filepath.Match(filter, r.Name)
				if !matched {
					continue
				}
			}
			id := strings.ToLower(r.Name)
			id = strings.ReplaceAll(id, ".", "-")
			allRepos = append(allRepos, importEntry{
				ID: id,
				Project: config.Project{
					Name:         r.Name,
					GitHubEvents: config.StringList{r.FullName},
				},
			})
		}

		page++
	}

	return allRepos, nil
}

func (h *handlers) handleValidateProjects(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projects := h.cfg.ResolveProjects()
	if projects == nil {
		return errorResult("no projects configured"), nil
	}

	projectFilter := req.GetString("project", "")
	if projectFilter != "" {
		p, ok := projects[projectFilter]
		if !ok {
			return errorResult(fmt.Sprintf("project %q not found", projectFilter)), nil
		}
		projects = map[string]config.Project{projectFilter: p}
	}

	token := os.Getenv("GITHUB_TOKEN")
	client := &http.Client{Timeout: 10 * time.Second}
	opts := config.ValidationOptions{
		Client:  client,
		Timeout: 10 * time.Second,
		Token:   token,
	}

	type validationResult struct {
		Project string `json:"project"`
		Source  string `json:"source"`
		Value   string `json:"value"`
		OK      bool   `json:"ok"`
		Error   string `json:"error,omitempty"`
	}

	var results []validationResult
	for id, proj := range projects {
		vResults := config.ValidateProject(ctx, opts, id, proj)
		for _, r := range vResults {
			vr := validationResult{
				Project: id,
				Source:  r.Source,
				Value:   r.Value,
				OK:      r.OK,
			}
			if r.Error != "" {
				vr.Error = r.Error
			}
			results = append(results, vr)
		}
	}

	return jsonResult(results), nil
}

func (h *handlers) handleExport(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	directory, err := req.RequireString("directory")
	if err != nil {
		return errorResult(err.Error()), nil
	}

	format := req.GetString("format", "parquet")
	source := req.GetString("source", "")
	project := req.GetString("project", "")

	written, err := store.Export(store.ExportOptions{
		DataDir:  h.cfg.DataDir(),
		OutDir:   directory,
		Format:   format,
		Source:   source,
		Project:  project,
		Projects: h.projectInfos(),
	})
	if err != nil {
		return errorResult(fmt.Sprintf("export: %v", err)), nil
	}

	var files []string
	for _, path := range written {
		files = append(files, filepath.Base(path))
	}

	return textResult(fmt.Sprintf("Exported %d files to %s: %s", len(files), directory, strings.Join(files, ", "))), nil
}

func (h *handlers) handleMigrate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dataDir := h.cfg.DataDir()

	current, err := store.SchemaVersion(dataDir)
	if err != nil {
		return errorResult(fmt.Sprintf("check schema version: %v", err)), nil
	}

	force := false
	if args := req.GetArguments(); args != nil {
		if v, ok := args["force"]; ok {
			force, _ = v.(bool)
		}
	}

	if force {
		current = 0
	}

	if current >= store.LatestSchemaVersion {
		return textResult(fmt.Sprintf("Data is already at schema version %d (latest)", current)), nil
	}

	applied, err := store.MigrateFrom(dataDir, current)
	if err != nil {
		return errorResult(fmt.Sprintf("migration failed after %d step(s): %v", applied, err)), nil
	}

	return textResult(fmt.Sprintf("Applied %d migration(s), now at schema version %d", applied, store.LatestSchemaVersion)), nil
}

func (h *handlers) handleListViews(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	viewsDir := h.cfg.ViewsDir()
	allViews, err := views.Discover(viewsDir, h.cfg.Views.Items, h.cfg.ViewsSource())
	if err != nil {
		return errorResult(fmt.Sprintf("discover views: %v", err)), nil
	}

	type viewEntry struct {
		Name      string `json:"name"`
		Framework string `json:"framework"`
		Source    string `json:"source"`
		Path      string `json:"path"`
		Output    string `json:"output"`
	}

	entries := make([]viewEntry, len(allViews))
	for i, v := range allViews {
		entries[i] = viewEntry{
			Name:      v.Name,
			Framework: string(v.Framework),
			Source:    v.Source,
			Path:      v.Path,
			Output:    v.Output,
		}
	}

	return jsonResult(entries), nil
}

func (h *handlers) handleShowView(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return errorResult(err.Error()), nil
	}

	viewsDir := h.cfg.ViewsDir()
	allViews, err := views.Discover(viewsDir, h.cfg.Views.Items, h.cfg.ViewsSource())
	if err != nil {
		return errorResult(fmt.Sprintf("discover views: %v", err)), nil
	}

	v, found := views.FindView(allViews, name)
	if !found {
		return errorResult(fmt.Sprintf("view %q not found", name)), nil
	}

	type viewDetail struct {
		Name      string `json:"name"`
		Framework string `json:"framework"`
		Source    string `json:"source"`
		Path      string `json:"path"`
		Output    string `json:"output"`
		Venv      string `json:"venv,omitempty"`
		Renderer  string `json:"renderer,omitempty"`
	}

	detail := viewDetail{
		Name:      v.Name,
		Framework: string(v.Framework),
		Source:    v.Source,
		Path:      v.Path,
		Output:    v.Output,
		Venv:      v.Venv,
	}

	if ver, err := views.CheckRenderer(v.Framework, v.Venv); err == nil {
		detail.Renderer = ver
	}

	return jsonResult(detail), nil
}

func (h *handlers) handleAddView(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return errorResult(err.Error()), nil
	}
	framework, err := req.RequireString("framework")
	if err != nil {
		return errorResult(err.Error()), nil
	}

	fw, err := views.ParseFramework(framework)
	if err != nil {
		return errorResult(err.Error()), nil
	}

	source := req.GetString("source", h.cfg.ViewsSource())
	if source != "parquet" && source != "jsonl" {
		return errorResult(fmt.Sprintf("invalid source %q (use parquet or jsonl)", source)), nil
	}

	viewsDir := h.cfg.ViewsDir()
	path, err := views.Scaffold(viewsDir, name, fw, source, h.cfg.DataDir())
	if err != nil {
		return errorResult(fmt.Sprintf("scaffold: %v", err)), nil
	}

	return textResult(fmt.Sprintf("Created view '%s' at %s", name, path)), nil
}

func (h *handlers) handleRemoveView(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return errorResult(err.Error()), nil
	}

	viewsDir := h.cfg.ViewsDir()
	allViews, err := views.Discover(viewsDir, h.cfg.Views.Items, h.cfg.ViewsSource())
	if err != nil {
		return errorResult(fmt.Sprintf("discover views: %v", err)), nil
	}

	v, found := views.FindView(allViews, name)
	if !found {
		return errorResult(fmt.Sprintf("view %q not found", name)), nil
	}

	if err := os.Remove(v.Path); err != nil && !os.IsNotExist(err) {
		return errorResult(fmt.Sprintf("remove source: %v", err)), nil
	}

	keepOutput := false
	if args := req.GetArguments(); args != nil {
		if val, ok := args["keep_output"]; ok {
			keepOutput, _ = val.(bool)
		}
	}

	if !keepOutput {
		os.Remove(v.Output)
	}

	return textResult(fmt.Sprintf("Removed view '%s'", name)), nil
}

func (h *handlers) handleRenderView(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return errorResult(err.Error()), nil
	}

	viewsDir := h.cfg.ViewsDir()
	allViews, err := views.Discover(viewsDir, h.cfg.Views.Items, h.cfg.ViewsSource())
	if err != nil {
		return errorResult(fmt.Sprintf("discover views: %v", err)), nil
	}

	v, found := views.FindView(allViews, name)
	if !found {
		return errorResult(fmt.Sprintf("view %q not found", name)), nil
	}

	if _, err := views.CheckRenderer(v.Framework, v.Venv); err != nil {
		return errorResult(err.Error()), nil
	}

	noExport := false
	if args := req.GetArguments(); args != nil {
		if val, ok := args["no_export"]; ok {
			noExport, _ = val.(bool)
		}
	}

	if !noExport && v.Source == "parquet" {
		dataDir := filepath.Join(viewsDir, "_data")
		if _, err := store.Export(store.ExportOptions{
			DataDir:  h.cfg.DataDir(),
			OutDir:   dataDir,
			Format:   "parquet",
			Projects: h.projectInfos(),
		}); err != nil {
			return errorResult(fmt.Sprintf("export data: %v", err)), nil
		}
	}

	if err := views.Render(v); err != nil {
		return errorResult(fmt.Sprintf("render: %v", err)), nil
	}

	return textResult(fmt.Sprintf("Rendered '%s' → %s", v.Name, v.Output)), nil
}

func (h *handlers) handleRenderViews(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	viewsDir := h.cfg.ViewsDir()
	allViews, err := views.Discover(viewsDir, h.cfg.Views.Items, h.cfg.ViewsSource())
	if err != nil {
		return errorResult(fmt.Sprintf("discover views: %v", err)), nil
	}

	prefix := req.GetString("prefix", "")
	noExport := false
	if args := req.GetArguments(); args != nil {
		if val, ok := args["no_export"]; ok {
			noExport, _ = val.(bool)
		}
	}

	var toRender []views.View
	for _, v := range allViews {
		if prefix == "" || strings.HasPrefix(v.Name, prefix) {
			toRender = append(toRender, v)
		}
	}

	if len(toRender) == 0 {
		return textResult("No views found to render"), nil
	}

	if !noExport && views.AnyUsesParquet(toRender) {
		dataDir := filepath.Join(viewsDir, "_data")
		if _, err := store.Export(store.ExportOptions{
			DataDir:  h.cfg.DataDir(),
			OutDir:   dataDir,
			Format:   "parquet",
			Projects: h.projectInfos(),
		}); err != nil {
			return errorResult(fmt.Sprintf("export data: %v", err)), nil
		}
	}

	var rendered int
	var errors []string
	for _, v := range toRender {
		if _, err := views.CheckRenderer(v.Framework, v.Venv); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", v.Name, err))
			continue
		}
		if err := views.Render(v); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", v.Name, err))
			continue
		}
		rendered++
	}

	msg := fmt.Sprintf("Rendered %d/%d views", rendered, len(toRender))
	if len(errors) > 0 {
		msg += "\nErrors:\n" + strings.Join(errors, "\n")
	}
	return textResult(msg), nil
}

func toStringList(s string) config.StringList {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var result config.StringList
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
