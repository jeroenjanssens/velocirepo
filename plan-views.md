# Plan: Views Feature

## Overview

Add a "views" feature to velocirepo that lets users scaffold, render, and serve dashboards, reports, notebooks, and images from their metrics data. Views support multiple frameworks (Quarto, Jupyter, Marimo, R scripts, SQL via ggsql) without locking users into any one tool. Velocirepo provides the infrastructure (scaffolding, rendering orchestration, staleness detection, dependency checks) while delegating actual rendering to the user's preferred framework.

## Terminology

- **View**: A source file (`.qmd`, `.ipynb`, `.py`, `.R`, `.sql`) that produces rendered output (HTML, PDF, PNG, etc.) from velocirepo metrics data.
- **Renderer**: The external tool that processes a view (e.g., `quarto`, `jupyter`, `python`, `Rscript`, `ggsql`).

## Config

### `velocirepo.toml` changes

Add a top-level `[views]` section for global settings, and a `[[views.items]]` array for per-view config:

```toml
[views]
dir = "velocirepo/views"   # default, relative to config dir

[[views.items]]
path = "weekly/stars.qmd"  # relative to views.dir
output = "docs/stars.html" # optional, relative to config dir; default: _output/ mirroring path structure

[[views.items]]
path = "downloads.py"
venv = ".venv"             # path to Python venv (relative to config dir)

[[views.items]]
path = "cran-trend.R"
# uses system Rscript by default
```

**Design decisions:**
- `dir` defaults to `"velocirepo/views"` if the `[views]` section is absent or `dir` is omitted.
- Views NOT listed in `[[views.items]]` still work with defaults (convention over configuration). Directory scanning finds all supported files in `dir`; config entries are only needed for overrides.
- Subdirectories within `views.dir` are fully supported. The view name/ID is the relative path without extension (e.g., `weekly/stars`).
- `venv` specifies a Python virtual environment path. Velocirepo activates it before running the renderer. If omitted, uses whatever `python`/`marimo`/`jupyter` is on PATH.
- `output` specifies where rendered output goes. Default: `{views.dir}/_output/{path-with-extension-replaced}`.

### Config struct changes (`internal/config/config.go`)

```go
type ViewsConfig struct {
    Dir   string     `toml:"dir"`
    Items []ViewItem `toml:"items"`
}

type ViewItem struct {
    Path   string `toml:"path"`
    Output string `toml:"output"`
    Venv   string `toml:"venv"`
}

type Config struct {
    Data     DataConfig         `toml:"data"`
    Settings SettingsConfig     `toml:"settings"`
    Views    ViewsConfig        `toml:"views"`
    Projects map[string]Project `toml:"projects"`
    // ...
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
```

## Supported Frameworks

| Extension | Renderer binary | Render command | Output |
|-----------|----------------|----------------|--------|
| `.qmd` | `quarto` | `quarto render {file} --output-dir {output_dir}` | HTML/PDF (per file config) |
| `.ipynb` | `jupyter` | `jupyter execute {file}` then `jupyter nbconvert --to html {file} --output {output}` | HTML |
| `.py` | `python` | `python {file}` (marimo files are executable by Python directly) | User-defined (script writes its own output) |
| `.R` | `Rscript` | `Rscript {file}` (script writes its own output) | User-defined |
| `.sql` | `ggsql` | `ggsql {file} -o {output}` | PNG/SVG |

Note: Marimo notebooks are valid Python files. They can be executed directly with `python file.py`. The `marimo` CLI is only needed for `serve-view` (interactive editing via `marimo edit`).

## CLI Commands

All commands are flat (no subcommand nesting), following the existing project pattern. Add a new command group:

```go
rootCmd.AddGroup(&cobra.Group{ID: "view", Title: "Views:"})
```

### `add-view [name]`

Scaffold a new view.

```
velocirepo add-view weekly/stars --framework quarto
velocirepo add-view overview --framework marimo
velocirepo add-view downloads --framework r
velocirepo add-view cran-badge --framework sql
```

**Flags:**
- `--framework` / `-f`: Required. One of: `quarto`, `jupyter`, `marimo`, `r`, `sql`.
- `--venv`: Optional. Python venv path to record in config.
- `--output` / `-o`: Optional. Custom output path.

**Behavior:**
1. Check that the renderer binary is installed. If not, print a warning with install instructions but still create the file (user may install later).
2. Create the source file at `{views.dir}/{name}.{ext}` with a template that:
   - Connects to velocirepo data (e.g., `import duckdb; con = duckdb.connect(); con.execute("SELECT * FROM read_json_auto('velocirepo/data/...')")`)
   - Includes a sample query relevant to the project's configured sources
   - Has a basic chart or table placeholder
3. Add an entry to `[[views.items]]` in `velocirepo.toml` only if non-default options are specified (venv, output).
4. Create the parent directory if needed.
5. Print confirmation: "Created view 'weekly/stars' at velocirepo/views/weekly/stars.qmd"

**Templates** should live in `internal/views/templates/` as embedded Go files. One template per framework.

### `remove-view <name>`

Remove a view source file and its config entry.

```
velocirepo remove-view weekly/stars
```

**Flags:**
- `--keep-output`: Don't delete rendered output.

**Behavior:**
1. Delete source file from `views.dir`.
2. Remove matching `[[views.items]]` entry from config (if present).
3. Delete rendered output (unless `--keep-output`).
4. Print confirmation.

### `list-views`

List all views with their status.

```
velocirepo list-views
```

**Output (table):**
```
NAME             FRAMEWORK  STATUS   OUTPUT
weekly/stars     quarto     stale    _output/weekly/stars.html
downloads        marimo     fresh    _output/downloads.html
cran-badge       sql        never    (not rendered yet)
overview         marimo     fresh    docs/overview.html
```

**Status logic:**
- `fresh`: output exists and is newer than both the source file and the most recent data file.
- `stale`: output exists but source or data has been modified since.
- `never`: no output exists.
- `error`: last render failed (check for `.render-error` marker or similar).

### `show-view <name>`

Show details about a specific view.

```
velocirepo show-view weekly/stars
```

**Output:**
```
Name:       weekly/stars
Framework:  quarto
Source:     velocirepo/views/weekly/stars.qmd
Output:     velocirepo/views/_output/weekly/stars.html
Venv:       (none)
Status:     stale (data updated 2026-06-21, output from 2026-06-19)
Renderer:   quarto 1.6.39
```

### `render-view [name]`

Render a single view.

```
velocirepo render-view weekly/stars
velocirepo render-view downloads --force
```

**Flags:**
- `--force`: Re-render even if fresh.

**Behavior:**
1. Resolve the view (find source, determine framework, find config overrides).
2. Check renderer is installed. Error if not (with install instructions).
3. If `venv` is configured, activate it (prepend `{venv}/bin` to PATH for the subprocess).
4. Run the render command.
5. Move/verify output is in the expected location.
6. Print: "Rendered 'weekly/stars' → velocirepo/views/_output/weekly/stars.html"

### `render-views`

Render all (or a subset of) views.

```
velocirepo render-views
velocirepo render-views --stale-only
velocirepo render-views weekly/
```

**Args:**
- Optional path prefix to filter (e.g., `weekly/` renders all views in that subtree).

**Flags:**
- `--stale-only`: Only render views whose status is `stale` or `never`.
- `--force`: Re-render all regardless of status.
- `--parallel` / `-j`: Number of concurrent renders (default: 1, sequential).

**Behavior:**
1. Discover all views (scan directory + merge config).
2. Filter by prefix/staleness.
3. Render each, collecting results.
4. Print summary: "Rendered 3/5 views (2 already fresh)"

### `serve-view <name>`

Start a dev server with live reload (delegates to the framework's own dev server).

```
velocirepo serve-view weekly/stars
```

**Behavior per framework:**
- `.qmd` → `quarto preview {file}`
- `.py` → `marimo edit {file}` (opens the marimo interactive editor; requires `marimo` to be installed)
- `.ipynb` → `jupyter notebook {file}`
- `.R` → Render and open output in browser (`open {output}`)
- `.sql` → Render and open output in browser (`open {output}`)

**Flags:**
- `--port`: Override port (passed through to underlying server).

## MCP Tools

Expose the following tools via the MCP server (in `cmd/velocirepo/cmd/mcp.go`):

| Tool name | Description |
|-----------|-------------|
| `add_view` | Scaffold a new view |
| `remove_view` | Remove a view |
| `list_views` | List all views with status |
| `show_view` | Show view details |
| `render_view` | Render a single view |
| `render_views` | Render all or filtered views |

(No `serve_view` via MCP — it's interactive only.)

## Internal Package: `internal/views/`

```
internal/views/
  views.go          # Core types, discovery, staleness logic
  render.go         # Render orchestration (dispatch to framework)
  templates/        # Embedded scaffold templates
    quarto.qmd.tmpl
    jupyter.ipynb.tmpl
    marimo.py.tmpl
    r.R.tmpl
    sql.sql.tmpl
```

### Key types

```go
type Framework string

const (
    FrameworkQuarto  Framework = "quarto"
    FrameworkJupyter Framework = "jupyter"
    FrameworkMarimo  Framework = "marimo"
    FrameworkR       Framework = "r"
    FrameworkSQL     Framework = "sql"
)

type View struct {
    Name      string    // relative path without extension, e.g. "weekly/stars"
    Path      string    // absolute path to source file
    Framework Framework
    Output    string    // absolute path to expected output
    Venv      string    // absolute path to venv (empty if none)
}

type Status string

const (
    StatusFresh Status = "fresh"
    StatusStale Status = "stale"
    StatusNever Status = "never"
    StatusError Status = "error"
)

// Discover scans the views directory and merges with config.
func Discover(viewsDir string, items []config.ViewItem) ([]View, error)

// GetStatus checks staleness by comparing timestamps.
func GetStatus(view View, dataDir string) Status

// Render executes the appropriate renderer for a view.
func Render(view View) error

// CheckRenderer verifies the renderer binary is available and returns version info.
func CheckRenderer(fw Framework, venv string) (version string, err error)
```

### Staleness detection

Compare three timestamps:
1. Source file mtime
2. Most recent data file mtime (latest `.jsonl` in the data dir)
3. Output file mtime

If output mtime < max(source mtime, data mtime), the view is stale.

### Dependency checking

Before rendering, run `{binary} --version` (or equivalent). Map of checks:

```go
var rendererChecks = map[Framework]struct {
    Binary     string
    VersionCmd []string
    InstallURL string
}{
    FrameworkQuarto:  {"quarto", []string{"quarto", "--version"}, "https://quarto.org/docs/get-started/"},
    FrameworkJupyter: {"jupyter", []string{"jupyter", "--version"}, "https://jupyter.org/install"},
    FrameworkMarimo:  {"python", []string{"python", "--version"}, "https://www.python.org/downloads/"},
    FrameworkR:       {"Rscript", []string{"Rscript", "--version"}, "https://cran.r-project.org/"},
    FrameworkSQL:     {"ggsql", []string{"ggsql", "--version"}, "https://github.com/jeroenjanssens/ggsql"},
}

// For serve-view, marimo requires an additional check:
var serveChecks = map[Framework]struct {
    Binary     string
    VersionCmd []string
    InstallURL string
}{
    FrameworkMarimo: {"marimo", []string{"marimo", "--version"}, "https://docs.marimo.io/getting_started/"},
}
```

If `venv` is set, look for the binary inside `{venv}/bin/` first.

## Scaffold Templates

Each template should produce a working view that connects to the data. Templates receive:
- `DataDir`: path to the data directory
- `Projects`: list of configured project IDs
- `ViewName`: name of the view

### Example: `quarto.qmd.tmpl`

```qmd
---
title: "{{.ViewName}}"
format: html
---

```{python}
import duckdb

con = duckdb.connect()
con.sql("""
    SELECT date, metric, value
    FROM read_json_auto('{{.DataDir}}/**/*.jsonl')
    WHERE metric = 'total_views'
    ORDER BY date
    LIMIT 100
""").show()
```
```

### Example: `marimo.py.tmpl`

```python
import marimo

app = marimo.App()

@app.cell
def _():
    import duckdb
    con = duckdb.connect()
    df = con.sql("""
        SELECT date, metric, value
        FROM read_json_auto('{{.DataDir}}/**/*.jsonl')
        WHERE metric = 'total_views'
        ORDER BY date
        LIMIT 100
    """).df()
    df

app.run()
```

### Example: `r.R.tmpl`

```r
library(duckdb)

con <- dbConnect(duckdb())
df <- dbGetQuery(con, "
    SELECT date, metric, value
    FROM read_json_auto('{{.DataDir}}/**/*.jsonl')
    WHERE metric = 'total_views'
    ORDER BY date
    LIMIT 100
")
print(df)
dbDisconnect(con, shutdown = TRUE)
```

### Example: `sql.sql.tmpl`

```sql
SELECT date, metric, SUM(value) as total
FROM read_json_auto('{{.DataDir}}/**/*.jsonl')
WHERE metric = 'total_views'
GROUP BY date, metric
ORDER BY date
```

## GitHub Actions Integration

### Render step in workflow

Users add to their existing velocirepo workflow:

```yaml
- name: Render views
  run: velocirepo render-views --stale-only

- name: Commit rendered output
  run: |
    git add velocirepo/views/_output/
    git diff --staged --quiet || git commit -m "Update rendered views"
```

Or deploy to GitHub Pages:

```yaml
- name: Deploy to Pages
  uses: actions/upload-pages-artifact@v3
  with:
    path: velocirepo/views/_output/
```

### CI install consideration

The `ci-install` command should be aware of views and install required renderers. This is a follow-up feature — for now, users manage renderer installation in their workflow.

## Output Directory

Default output structure mirrors source structure:

```
velocirepo/views/
  _output/                 # default output dir (gitignored or committed)
    weekly/
      stars.html
    downloads.html
    cran-badge.png
  weekly/
    stars.qmd
  downloads.py
  cran-badge.sql
```

Add `_output/` to a default `.gitignore` inside the views dir (created by `add-view` on first run). Users can override output paths per-view in config or commit the outputs if they prefer.

## README Updates

Update `README.md` to document the views feature. Add a new section (after the existing Querying/Badges sections) with:

### Section: Views

**Intro paragraph** explaining that views let you create dashboards, reports, notebooks, and images from your metrics data using your preferred framework.

**Quick-start example:**

```bash
# Scaffold a SQL view
velocirepo add-view cran-downloads --framework sql

# Edit the generated file (velocirepo/views/cran-downloads.sql)
# Then render it to an image
velocirepo render-view cran-downloads

# Open the output
open velocirepo/views/_output/cran-downloads.png
```

**Supported frameworks table:**

| Framework | Extension | Renderer | Serve mode |
|-----------|-----------|----------|------------|
| Quarto | `.qmd` | `quarto` | `quarto preview` |
| Jupyter | `.ipynb` | `jupyter` | `jupyter notebook` |
| Marimo | `.py` | `python` | `marimo edit` |
| R | `.R` | `Rscript` | render + open |
| SQL | `.sql` | `ggsql` | render + open |

**CLI commands table:**

| Command | Description |
|---------|-------------|
| `add-view` | Scaffold a new view |
| `remove-view` | Remove a view and its output |
| `list-views` | List all views with status (fresh/stale/never) |
| `show-view` | Show details about a view |
| `render-view` | Render a single view |
| `render-views` | Render all or filtered views |
| `serve-view` | Start dev server or open rendered output |

**Configuration example:**

```toml
[views]
dir = "velocirepo/views"

[[views.items]]
path = "cran-downloads.sql"

[[views.items]]
path = "overview.py"
venv = ".venv"
```

**CI usage example:**

```yaml
- name: Render views
  run: velocirepo render-views --stale-only

- name: Commit rendered output
  run: |
    git add velocirepo/views/_output/
    git diff --staged --quiet || git commit -m "Update rendered views"
```

## Implementation Order

1. **Config**: Add `ViewsConfig` and `ViewItem` to `internal/config/config.go`. Add `ViewsDir()` method to `Config`.
2. **Core package**: Create `internal/views/` with types, discovery, staleness, and renderer dispatch.
3. **Templates**: Create embedded templates in `internal/views/templates/`.
4. **CLI commands** (in order):
   - `add-view` (scaffold + config write)
   - `list-views` (discovery + staleness)
   - `show-view`
   - `render-view` (single render)
   - `render-views` (batch render)
   - `remove-view`
   - `serve-view`
5. **MCP tools**: Expose view commands via MCP server.
6. **README**: Add views documentation section.
7. **Tests**: Unit tests for discovery, staleness logic, config parsing. Integration tests that scaffold and render a simple view.

## File Listing (new files to create)

```
internal/views/views.go
internal/views/render.go
internal/views/templates/quarto.qmd.tmpl
internal/views/templates/jupyter.ipynb.tmpl
internal/views/templates/marimo.py.tmpl
internal/views/templates/r.R.tmpl
internal/views/templates/sql.sql.tmpl
cmd/velocirepo/cmd/view_add.go
cmd/velocirepo/cmd/view_remove.go
cmd/velocirepo/cmd/view_list.go
cmd/velocirepo/cmd/view_show.go
cmd/velocirepo/cmd/view_render.go
cmd/velocirepo/cmd/view_render_all.go
cmd/velocirepo/cmd/view_serve.go
```

## Files to modify

```
internal/config/config.go       # Add ViewsConfig, ViewItem structs
cmd/velocirepo/cmd/root.go      # Add "view" group and register commands
cmd/velocirepo/cmd/mcp.go       # Add view-related MCP tools
README.md                       # Add views documentation section
```
