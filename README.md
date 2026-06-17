# velocirepo

velocirepo fetches and aggregates metrics for your open-source projects, building a historical record you can query and commit to git. It currently supports the following sources:

- **GitHub** — cumulative totals (stars, forks, open issues, open PRs, comments)
- **GitHub Events** — daily activity counts (stars, forks, issues, PRs, pushes, releases, comments, reviews)
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
github-events = "owner/repo"
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
velocirepo fetch github          Fetch GitHub metrics (stars, forks, issues, PRs, comments)
velocirepo fetch github-traffic  Fetch GitHub traffic (views and clones)
velocirepo fetch github-events   Fetch GitHub event activity
velocirepo fetch pypi            Fetch PyPI download counts
velocirepo fetch cran            Fetch CRAN download counts
velocirepo fetch homebrew        Fetch Homebrew install counts
velocirepo fetch plausible       Fetch Plausible analytics
velocirepo fetch openvsx         Fetch Open VSX install counts
velocirepo fetch all             Fetch from all configured sources

velocirepo export -o FILE    Export all metrics to Parquet or CSV

velocirepo query run [sql]   Run a SQL query against the metrics data
velocirepo query schema      Show the metrics table schema

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

velocirepo stores all fetched data as JSONL files. You can query them directly using SQL (powered by DuckDB). Three views are available: `metrics`, `github_events`, and `projects`.

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

## Data storage

Metrics are stored as JSONL files at `velocirepo/data/<source>/<project-id>/<date>.jsonl`. Daily files are automatically aggregated into monthly and yearly files once the period is complete.

You can either keep metrics in the same repository as your code, or create a dedicated metrics repository. A separate repo is useful when you want to track multiple projects in one place or keep metric history out of your main codebase.

The `query` command reads JSONL files directly using DuckDB.

## License

MIT
