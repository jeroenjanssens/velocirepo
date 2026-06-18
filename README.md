# velocirepo

velocirepo fetches and aggregates metrics for your open-source projects, building a historical record you can query and commit to git. It currently supports the following sources:

- **GitHub** — individual events (stars, forks, issues, PRs) with user and timestamp
- **GitHub Traffic** — daily page views and git clones (requires admin access)
- **PyPI** — daily download counts
- **CRAN** — daily download counts
- **Homebrew** — install counts
- **Plausible** — pageviews, visitors, visits
- **OpenVSX** — downloads, reviews, ratings

## Configuration

Create a `velocirepo.toml` in your project root:

```toml
[projects.my-project]
name = "My Project"
github = "owner/repo"
github-traffic = "owner/repo"
pypi = "my-package"

[projects.other-project]
name = "Other Project"
github = ["owner/other", "owner/other-utils"]
cran = "other"
homebrew = "other"
```

Each source field accepts either a single string or an array of strings, so you can track multiple repositories or packages under one project.

The `github-traffic` source fetches daily page views and clone counts. GitHub only retains this data for 14 days, so velocirepo preserves it before it's lost. It requires a token with **Administration:read** permission (or the `repo` scope for classic tokens).

Or initialize one interactively (auto-detects sources from your repository):

```bash
velocirepo project init
```

velocirepo looks for `velocirepo.toml` by walking up from the current directory. Override with `--config` or the `VELOCIREPO_CONFIG` environment variable.

### Environment variables

| Variable | Description |
|----------|-------------|
| `GITHUB_TOKEN` | GitHub personal access token (increases rate limits) |
| `PLAUSIBLE_KEY` | Plausible API key |
| `VELOCIREPO_CONFIG` | Path to config file |

These can also be set in a `.env` file in the current directory.

## Installation

velocirepo can be used as a GitHub Action for automated nightly fetching, or installed locally for ad-hoc use.

### GitHub Actions

```yaml
name: Fetch Metrics

on:
  schedule:
    - cron: '0 1 * * *'
  workflow_dispatch:

permissions:
  contents: write

jobs:
  fetch:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: jeroenjanssens/velocirepo@v0
        with:
          github-token: ${{ secrets.GH_TOKEN }}
          # plausible-key: ${{ secrets.PLAUSIBLE_KEY }}

      - name: Commit and push
        run: |
          git config --local user.email "github-actions[bot]@users.noreply.github.com"
          git config --local user.name "github-actions[bot]"
          git add velocirepo/
          git diff --staged --quiet || git commit -m "Update metrics - $(date -u +'%Y-%m-%d')"
          git push
```

Or generate this workflow automatically (no installation required):

```bash
uvx velocirepo ci install
```

### Local installation

#### Homebrew (macOS / Linux)

```bash
brew install jeroenjanssens/tap/velocirepo
```

#### Scoop (Windows)

```powershell
scoop bucket add jeroenjanssens https://github.com/jeroenjanssens/scoop-bucket
scoop install velocirepo
```

#### uv

```bash
uv tool install velocirepo
```

Or run without installing:

```bash
uvx velocirepo fetch all
```

#### Go

```bash
go install github.com/jeroenjanssens/velocirepo/cmd/velocirepo@latest
```

#### From source

```bash
git clone https://github.com/jeroenjanssens/velocirepo.git
cd velocirepo
go build -o bin/velocirepo ./cmd/velocirepo
cp bin/velocirepo ~/.local/bin/
```

#### Shell installer (macOS / Linux)

```bash
curl -sSfL https://raw.githubusercontent.com/jeroenjanssens/velocirepo/main/install.sh | sh
```

This downloads the latest binary to `./bin/velocirepo`. Set `INSTALL_DIR` to install elsewhere:

```bash
curl -sSfL https://raw.githubusercontent.com/jeroenjanssens/velocirepo/main/install.sh | INSTALL_DIR=/usr/local/bin sh
```

#### Download binary

Pre-built binaries for Linux, macOS, and Windows are available on the [Releases](https://github.com/jeroenjanssens/velocirepo/releases) page.

## Usage

```
velocirepo fetch github          Fetch GitHub events (stars, forks, issues, PRs)
velocirepo fetch github-traffic  Fetch GitHub traffic (views and clones)
velocirepo fetch pypi            Fetch PyPI download counts
velocirepo fetch cran            Fetch CRAN download counts
velocirepo fetch homebrew        Fetch Homebrew install counts
velocirepo fetch plausible       Fetch Plausible analytics
velocirepo fetch openvsx         Fetch Open VSX install counts
velocirepo fetch all             Fetch from all configured sources

velocirepo export <dir>      Export data to Parquet or CSV files

velocirepo query run [sql]   Run a SQL query against the metrics data
velocirepo query schema      Show table schemas

velocirepo badge stars       Generate a stars badge
velocirepo badge forks       Generate a forks badge
velocirepo badge downloads   Generate a downloads badge
velocirepo badge pageviews   Generate a pageviews badge
velocirepo badge custom      Generate a badge from a custom query

velocirepo project init      Create a new velocirepo.toml
velocirepo project add       Add a project to the config
velocirepo project update    Update a project's configuration
velocirepo project remove    Remove a project from the config
velocirepo project rename    Rename a project's ID
velocirepo project list      List configured projects
velocirepo project show      Show project details
velocirepo project import    Bulk-import from GitHub org/user or file
velocirepo project validate  Validate source URLs

velocirepo ci install        Generate a GitHub Actions workflow

velocirepo version           Print version information
```

## Querying the data

velocirepo stores all fetched data as JSONL files. You can query them directly using SQL (powered by DuckDB). Four views are available:

- `github_events` — individual GitHub events with user and timestamp
- `github` — aggregated daily counts per project, repo, and event type (computed from `github_events`)
- `metrics` — time-series metrics from other sources (PyPI, CRAN, Homebrew, Plausible, OpenVSX, GitHub Traffic)
- `projects` — project metadata from your config

### Total stars per project from event history

```bash
velocirepo query run "
  SELECT p.name, COUNT(*) AS stars
  FROM github_events e
  JOIN projects p ON e.project = p.id
  WHERE e.event_type = 'star'
  GROUP BY p.name
  ORDER BY stars DESC
  LIMIT 5
"
```

```
┌─────────────┬───────┐
│    name     │ stars │
├─────────────┼───────┤
│ ggplot2     │ 6877  │
│ Shiny for R │ 5600  │
│ Quarto      │ 5274  │
│ dplyr       │ 4995  │
│ plotnine    │ 4500  │
└─────────────┴───────┘
```

### Monthly star activity for a project

```bash
velocirepo query run "
  SELECT date_trunc('month', datetime)::DATE AS month, COUNT(*) AS stars
  FROM github_events
  WHERE project = 'quarto' AND event_type = 'star'
  GROUP BY month
  ORDER BY month DESC
  LIMIT 5
"
```

```
┌────────────┬───────┐
│   month    │ stars │
├────────────┼───────┤
│ 2026-02-01 │ 40    │
│ 2026-01-01 │ 107   │
│ 2025-12-01 │ 91    │
│ 2025-11-01 │ 97    │
│ 2025-10-01 │ 85    │
└────────────┴───────┘
```

### Daily star counts using the aggregated view

The `github` view provides daily counts without needing to write `GROUP BY` yourself:

```bash
velocirepo query run "
  SELECT project, date, count
  FROM github
  WHERE event_type = 'star'
  ORDER BY date DESC
  LIMIT 5
"
```

### Latest cumulative metrics

```bash
velocirepo query run "
  SELECT project, metric, date, value
  FROM metrics
  ORDER BY date DESC
  LIMIT 5
"
```

```
┌───────────┬─────────────────┬────────────┬─────────┐
│  project  │     metric      │    date    │  value  │
├───────────┼─────────────────┼────────────┼─────────┤
│ databot   │ total_downloads │ 2026-06-16 │ 49504   │
│ databot   │ rating          │ 2026-06-16 │ 5       │
│ databot   │ reviews         │ 2026-06-16 │ 1       │
│ publisher │ total_downloads │ 2026-06-16 │ 1562247 │
│ publisher │ reviews         │ 2026-06-16 │ 0       │
└───────────┴─────────────────┴────────────┴─────────┘
```

### Output formats

By default, results are printed as a table. Use `--json` or `--csv` for machine-readable output:

```bash
velocirepo query run --csv "SELECT project, metric, value FROM metrics LIMIT 3"
velocirepo query run --json "SELECT project, metric, value FROM metrics LIMIT 3"
```

The `query schema` command shows all available columns:

```bash
velocirepo query schema
```

## Exporting data

Export metrics, events, and project metadata to Parquet or CSV for use in other tools:

```bash
velocirepo export ./out/
```

This writes one file per table:

```
  out/metrics.parquet (257 KB)
  out/github_events.parquet (2.9 MB)
  out/projects.parquet (2 KB)
```

Use `--format csv` for CSV output, and `--source` or `--project` to filter:

```bash
velocirepo export ./out/ --format csv
velocirepo export ./out/ --source github
velocirepo export ./out/ --project quarto
```

## Generating badges

Generate shields.io-style SVG badges from your metrics data:

```bash
velocirepo badge stars -o stars.svg
velocirepo badge forks --project quarto -o forks.svg
velocirepo badge downloads --project ggplot2 -o downloads.svg
```

For custom metrics, provide a SQL query that returns a single value:

```bash
velocirepo badge custom \
  --label "contributors" \
  --query "SELECT COUNT(DISTINCT \"user\") AS value FROM github_events" \
  --color "#ea7233" \
  -o contributors.svg
```

### Styles

Three styles are available via `--style`:

| Style | Description |
|-------|-------------|
| `flat` (default) | Subtle gradient, rounded corners |
| `flat-square` | No gradient, sharp corners |
| `plastic` | Heavy gradient, more rounded |

### Options

| Flag | Default | Description |
|------|---------|-------------|
| `--project` | all | Scope to a specific project |
| `--style` | `flat` | Badge style |
| `--color` | preset-dependent | Message background color (hex) |
| `--label-color` | `#555` | Label background color |
| `--label` | preset-dependent | Override label text |
| `--height` | style-dependent | Badge height in pixels |
| `--radius` | style-dependent | Corner radius (0 = square) |
| `-o` | stdout | Output file path |

Numbers are automatically formatted for readability (e.g., `5274` becomes `5.3k`, `1500000` becomes `1.5M`).

## Data storage

All data is stored as JSONL files at `velocirepo/data/<source>/<project-id>/<date>.jsonl`. Daily files are automatically aggregated into monthly and yearly files once the period is complete.

### Metrics sources

Sources like PyPI, CRAN, Homebrew, Plausible, OpenVSX, and GitHub Traffic store one JSON object per metric per day:

```json
{"source":"pypi","metric":"downloads","project_id":"plotnine","target":"plotnine","date":"2026-06-15","value":1523}
{"source":"openvsx","metric":"total_downloads","project_id":"quarto","target":"quarto/quarto","date":"2026-06-15","value":1250000}
```

Fields:

| Field | Description |
|-------|-------------|
| `source` | Source name (pypi, cran, homebrew, plausible, openvsx, github-traffic) |
| `metric` | Metric name (downloads, pageviews, views, clones, etc.) |
| `project_id` | Project ID from your config |
| `target` | Specific package, repo, site, or extension being tracked |
| `date` | Date of the measurement |
| `value` | Integer value |
| `tags` | Optional key-value metadata |

### GitHub source

The GitHub source stores individual events rather than aggregated counts, giving you full historical detail including who performed each action and when:

```json
{"source":"github","event_type":"star","project_id":"quarto","github_repo":"quarto-dev/quarto-cli","datetime":"2026-06-15T14:23:01Z","user":"alice"}
{"source":"github","event_type":"fork","project_id":"quarto","github_repo":"quarto-dev/quarto-cli","datetime":"2026-06-15T09:11:44Z","user":"bob"}
```

Fields:

| Field | Description |
|-------|-------------|
| `source` | Always `github` |
| `event_type` | One of: star, fork, issue_open, issue_close, pr_open, pr_merge |
| `project_id` | Project ID from your config |
| `github_repo` | The owner/repo being tracked |
| `datetime` | Full timestamp of the event |
| `user` | GitHub username who performed the action |

### DuckDB views

The `query` command reads JSONL files directly using DuckDB and exposes four views:

| View | Description |
|------|-------------|
| `github_events` | Raw GitHub events with all fields as stored on disk |
| `github` | Aggregated daily counts per project, target, and event type (computed from `github_events`) |
| `metrics` | Time-series metrics from all other sources |
| `projects` | Project metadata from your config |

The `github` view provides a convenient aggregation so you don't need to write `GROUP BY` clauses yourself. It uses `target` (aliased from `github_repo`) for consistency with the `metrics` view.

### Repository layout

You can either keep metrics in the same repository as your code, or create a dedicated metrics repository. A separate repo is useful when you want to track multiple projects in one place or keep metric history out of your main codebase.

## License

MIT
