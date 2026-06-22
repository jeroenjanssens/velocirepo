package mcp

import (
	"github.com/jeroenjanssens/velocirepo/internal/config"
	"github.com/jeroenjanssens/velocirepo/internal/version"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type ServerOptions struct {
	Config   *config.Config
	ReadOnly bool
}

func NewServer(opts ServerOptions) *server.MCPServer {
	s := server.NewMCPServer(
		"velocirepo",
		version.Version,
	)

	h := &handlers{cfg: opts.Config}

	s.AddTool(queryTool(), h.handleQuery)
	s.AddTool(schemaTool(), h.handleSchema)
	s.AddTool(listProjectsTool(), h.handleListProjects)
	s.AddTool(showProjectTool(), h.handleShowProject)
	s.AddTool(badgeTool(), h.handleBadge)
	s.AddTool(versionTool(), h.handleVersion)

	if !opts.ReadOnly {
		s.AddTool(fetchTool(), h.handleFetch)
		s.AddTool(fetchGitHubTool(), h.handleFetchGitHub)
		s.AddTool(fetchTrafficTool(), h.handleFetchTraffic)
		s.AddTool(fetchPyPITool(), h.handleFetchPyPI)
		s.AddTool(fetchCRANTool(), h.handleFetchCRAN)
		s.AddTool(fetchHomebrewTool(), h.handleFetchHomebrew)
		s.AddTool(fetchPlausibleTool(), h.handleFetchPlausible)
		s.AddTool(fetchOpenVSXTool(), h.handleFetchOpenVSX)
		s.AddTool(fetchYouTubeTool(), h.handleFetchYouTube)
		s.AddTool(addProjectTool(), h.handleAddProject)
		s.AddTool(updateProjectTool(), h.handleUpdateProject)
		s.AddTool(removeProjectTool(), h.handleRemoveProject)
		s.AddTool(renameProjectTool(), h.handleRenameProject)
		s.AddTool(importProjectsTool(), h.handleImportProjects)
		s.AddTool(validateProjectsTool(), h.handleValidateProjects)
		s.AddTool(exportTool(), h.handleExport)
		s.AddTool(migrateTool(), h.handleMigrate)
	}

	return s
}

func queryTool() mcp.Tool {
	return mcp.NewTool("query",
		mcp.WithDescription("Run a SQL query against the metrics data. Available views: metrics (unified time-series), github_events (raw events with user/timestamp), youtube_index (video metadata), projects (config metadata). Default LIMIT is 1000."),
		mcp.WithString("sql", mcp.Required(), mcp.Description("SQL query to execute")),
		mcp.WithNumber("limit", mcp.Description("Maximum rows to return (default: 1000)")),
	)
}

func schemaTool() mcp.Tool {
	return mcp.NewTool("schema",
		mcp.WithDescription("Show column definitions for all DuckDB views: metrics, github_events, youtube_index, projects."),
	)
}

func listProjectsTool() mcp.Tool {
	return mcp.NewTool("list_projects",
		mcp.WithDescription("List all configured projects with their source configurations."),
	)
}

func showProjectTool() mcp.Tool {
	return mcp.NewTool("show_project",
		mcp.WithDescription("Show detailed information about a project including per-source fetch stats."),
		mcp.WithString("id", mcp.Required(), mcp.Description("Project ID")),
	)
}

func badgeTool() mcp.Tool {
	return mcp.NewTool("badge",
		mcp.WithDescription("Generate an SVG badge from metrics data. Types: stars, forks, downloads, pageviews, custom."),
		mcp.WithString("type", mcp.Required(), mcp.Description("Badge type: stars, forks, downloads, pageviews, or custom")),
		mcp.WithString("project", mcp.Description("Scope to a specific project")),
		mcp.WithString("query", mcp.Description("SQL query returning a single value (required for custom type)")),
		mcp.WithString("label", mcp.Description("Override label text (required for custom type)")),
		mcp.WithString("style", mcp.Description("Badge style: flat, flat-square, or plastic")),
		mcp.WithString("color", mcp.Description("Message background color (hex)")),
	)
}

func versionTool() mcp.Tool {
	return mcp.NewTool("version",
		mcp.WithDescription("Show velocirepo version, commit hash, and build date."),
	)
}

func fetchTool() mcp.Tool {
	return mcp.NewTool("fetch",
		mcp.WithDescription("Fetch metrics from all configured sources for all projects."),
		mcp.WithString("project", mcp.Description("Fetch only this project ID")),
		mcp.WithString("start_date", mcp.Description("Start date (YYYY-MM-DD)")),
		mcp.WithString("end_date", mcp.Description("End date (YYYY-MM-DD, default: yesterday)")),
	)
}

func fetchGitHubTool() mcp.Tool {
	return mcp.NewTool("fetch_github",
		mcp.WithDescription("Fetch GitHub events (stars, forks, issues, PRs)."),
		mcp.WithString("project", mcp.Description("Fetch only this project ID")),
		mcp.WithString("start_date", mcp.Description("Start date (YYYY-MM-DD)")),
		mcp.WithString("end_date", mcp.Description("End date (YYYY-MM-DD, default: yesterday)")),
	)
}

func fetchTrafficTool() mcp.Tool {
	return mcp.NewTool("fetch_traffic",
		mcp.WithDescription("Fetch GitHub traffic data (views and clones). Requires GITHUB_TOKEN with admin access."),
		mcp.WithString("project", mcp.Description("Fetch only this project ID")),
		mcp.WithString("start_date", mcp.Description("Start date (YYYY-MM-DD)")),
		mcp.WithString("end_date", mcp.Description("End date (YYYY-MM-DD, default: yesterday)")),
	)
}

func fetchPyPITool() mcp.Tool {
	return mcp.NewTool("fetch_pypi",
		mcp.WithDescription("Fetch PyPI download statistics."),
		mcp.WithString("project", mcp.Description("Fetch only this project ID")),
		mcp.WithString("start_date", mcp.Description("Start date (YYYY-MM-DD)")),
		mcp.WithString("end_date", mcp.Description("End date (YYYY-MM-DD, default: yesterday)")),
	)
}

func fetchCRANTool() mcp.Tool {
	return mcp.NewTool("fetch_cran",
		mcp.WithDescription("Fetch CRAN download statistics."),
		mcp.WithString("project", mcp.Description("Fetch only this project ID")),
		mcp.WithString("start_date", mcp.Description("Start date (YYYY-MM-DD)")),
		mcp.WithString("end_date", mcp.Description("End date (YYYY-MM-DD, default: yesterday)")),
	)
}

func fetchHomebrewTool() mcp.Tool {
	return mcp.NewTool("fetch_homebrew",
		mcp.WithDescription("Fetch Homebrew install counts."),
		mcp.WithString("project", mcp.Description("Fetch only this project ID")),
		mcp.WithString("start_date", mcp.Description("Start date (YYYY-MM-DD)")),
		mcp.WithString("end_date", mcp.Description("End date (YYYY-MM-DD, default: yesterday)")),
	)
}

func fetchPlausibleTool() mcp.Tool {
	return mcp.NewTool("fetch_plausible",
		mcp.WithDescription("Fetch Plausible analytics (pageviews, visitors, visits). Requires PLAUSIBLE_TOKEN."),
		mcp.WithString("project", mcp.Description("Fetch only this project ID")),
		mcp.WithString("start_date", mcp.Description("Start date (YYYY-MM-DD)")),
		mcp.WithString("end_date", mcp.Description("End date (YYYY-MM-DD, default: yesterday)")),
	)
}

func fetchOpenVSXTool() mcp.Tool {
	return mcp.NewTool("fetch_openvsx",
		mcp.WithDescription("Fetch Open VSX extension metrics (downloads, reviews, ratings)."),
		mcp.WithString("project", mcp.Description("Fetch only this project ID")),
		mcp.WithString("start_date", mcp.Description("Start date (YYYY-MM-DD)")),
		mcp.WithString("end_date", mcp.Description("End date (YYYY-MM-DD, default: yesterday)")),
	)
}

func fetchYouTubeTool() mcp.Tool {
	return mcp.NewTool("fetch_youtube",
		mcp.WithDescription("Fetch YouTube metrics (views, likes, comments, subscribers). Requires YOUTUBE_TOKEN."),
		mcp.WithString("project", mcp.Description("Fetch only this project ID")),
		mcp.WithString("start_date", mcp.Description("Start date (YYYY-MM-DD)")),
		mcp.WithString("end_date", mcp.Description("End date (YYYY-MM-DD, default: yesterday)")),
	)
}

func addProjectTool() mcp.Tool {
	return mcp.NewTool("add_project",
		mcp.WithDescription("Add a new project to the velocirepo config."),
		mcp.WithString("id", mcp.Required(), mcp.Description("Project ID (lowercase alphanumeric with hyphens)")),
		mcp.WithString("name", mcp.Description("Display name (defaults to ID)")),
		mcp.WithString("github_events", mcp.Description("GitHub owner/repo for events")),
		mcp.WithString("github_traffic", mcp.Description("GitHub owner/repo for traffic")),
		mcp.WithString("pypi", mcp.Description("PyPI package name")),
		mcp.WithString("cran", mcp.Description("CRAN package name")),
		mcp.WithString("homebrew", mcp.Description("Homebrew formula")),
		mcp.WithString("plausible", mcp.Description("Plausible site ID")),
		mcp.WithString("openvsx", mcp.Description("OpenVSX extension (publisher/extension)")),
		mcp.WithString("youtube", mcp.Description("YouTube channel (@handle), playlist, or video ID")),
	)
}

func updateProjectTool() mcp.Tool {
	return mcp.NewTool("update_project",
		mcp.WithDescription("Update a project's configuration. Only specified fields are changed."),
		mcp.WithString("id", mcp.Required(), mcp.Description("Project ID to update")),
		mcp.WithString("name", mcp.Description("New display name")),
		mcp.WithString("github_events", mcp.Description("GitHub owner/repo for events (empty to remove)")),
		mcp.WithString("github_traffic", mcp.Description("GitHub owner/repo for traffic (empty to remove)")),
		mcp.WithString("pypi", mcp.Description("PyPI package name (empty to remove)")),
		mcp.WithString("cran", mcp.Description("CRAN package name (empty to remove)")),
		mcp.WithString("homebrew", mcp.Description("Homebrew formula (empty to remove)")),
		mcp.WithString("plausible", mcp.Description("Plausible site ID (empty to remove)")),
		mcp.WithString("openvsx", mcp.Description("OpenVSX extension (empty to remove)")),
		mcp.WithString("youtube", mcp.Description("YouTube target (empty to remove)")),
	)
}

func removeProjectTool() mcp.Tool {
	return mcp.NewTool("remove_project",
		mcp.WithDescription("Remove a project from the config."),
		mcp.WithString("id", mcp.Required(), mcp.Description("Project ID to remove")),
		mcp.WithBoolean("delete_data", mcp.Description("Also remove the project's data directories")),
	)
}

func renameProjectTool() mcp.Tool {
	return mcp.NewTool("rename_project",
		mcp.WithDescription("Rename a project ID (also moves data directories)."),
		mcp.WithString("old_id", mcp.Required(), mcp.Description("Current project ID")),
		mcp.WithString("new_id", mcp.Required(), mcp.Description("New project ID")),
	)
}

func importProjectsTool() mcp.Tool {
	return mcp.NewTool("import_projects",
		mcp.WithDescription("Bulk-import projects from a GitHub organization or user."),
		mcp.WithString("github_org", mcp.Description("Import repos from this GitHub organization")),
		mcp.WithString("github_user", mcp.Description("Import repos from this GitHub user")),
		mcp.WithString("filter", mcp.Description("Glob pattern to filter repo names")),
		mcp.WithBoolean("skip_existing", mcp.Description("Skip projects that already exist in config")),
	)
}

func validateProjectsTool() mcp.Tool {
	return mcp.NewTool("validate_projects",
		mcp.WithDescription("Verify that configured source URLs are reachable via HTTP HEAD checks."),
		mcp.WithString("project", mcp.Description("Validate only this project")),
	)
}

func exportTool() mcp.Tool {
	return mcp.NewTool("export",
		mcp.WithDescription("Export metrics data to Parquet or CSV files."),
		mcp.WithString("directory", mcp.Required(), mcp.Description("Output directory path")),
		mcp.WithString("format", mcp.Description("Output format: parquet or csv (default: parquet)")),
		mcp.WithString("source", mcp.Description("Export only this source")),
		mcp.WithString("project", mcp.Description("Export only this project")),
	)
}

func migrateTool() mcp.Tool {
	return mcp.NewTool("migrate",
		mcp.WithDescription("Migrate data to the latest schema version."),
		mcp.WithBoolean("force", mcp.Description("Re-run all migrations from scratch")),
	)
}
