# velocirepo

Track your open-source project's pulse across package registries, GitHub, and the web.

## Table of contents

- [Overview](#overview)
- [Installation](#installation)
- [Configuration](#configuration)
- [Usage](#usage)
- [Data storage](#data-storage)
- [Migrating data](#migrating-data)
- [Querying the data](#querying-the-data)
- [Exporting data](#exporting-data)
- [Using the data with other tools](#using-the-data-with-other-tools)
- [Indicators](#indicators)
- [Generating badges](#generating-badges)
- [Views](#views)
- [MCP server](#mcp-server)

## Overview

velocirepo collects metrics from multiple sources, stores them as JSONL files, and exposes them via SQL (powered by DuckDB). It's designed to run on a schedule (e.g., nightly via GitHub Actions) and commit the results to git, giving you a permanent record of your project's growth.

Supported sources:

| Source | What it tracks |
|--------|----------------|
| **GitHub Events** | Individual events (stars, forks, issues, PRs) with user and timestamp |
| **GitHub Traffic** | Daily page views and git clones (requires admin access) |
| **PyPI** | Daily download counts |
| **CRAN** | Daily download counts |
| **Homebrew** | Install counts (30-day, 90-day, 365-day, lifetime) |
| **Plausible** | Daily pageviews, visitors, visits |
| **OpenVSX** | Total downloads, reviews, ratings |
| **YouTube** | Views, likes, comments, subscribers (channel and per-video) |

## Installation

velocirepo can be used as a GitHub Action for automated nightly fetching, or installed locally for ad-hoc use.

### GitHub Actions (recommended)

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
          # plausible-token: ${{ secrets.PLAUSIBLE_TOKEN }}
          # youtube-token: ${{ secrets.YOUTUBE_TOKEN }}

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
uvx velocirepo install-ci
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
uvx velocirepo fetch
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

### Shell completion

```bash
# Zsh (add to ~/.zshrc)
eval "$(velocirepo completion zsh)"

# Bash (add to ~/.bashrc)
eval "$(velocirepo completion bash)"

# Fish
velocirepo completion fish | source
```

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
youtube = "@ChannelHandle"
```

Each source field accepts either a single string or an array of strings, so you can track multiple repositories or packages under one project.

The `github-traffic` source fetches daily page views and clone counts. GitHub only retains this data for 14 days, so velocirepo preserves it before it's lost. It requires a token with **Administration:read** permission (or the `repo` scope for classic tokens).

Or initialize one interactively (auto-detects sources from your repository):

```bash
velocirepo init
```

velocirepo looks for `velocirepo.toml` by walking up from the current directory. Override with `--config` or the `VELOCIREPO_CONFIG` environment variable.

### Environment variables

| Variable | Description |
|----------|-------------|
| `GITHUB_TOKEN` | GitHub personal access token (increases rate limits) |
| `PLAUSIBLE_TOKEN` | Plausible API token |
| `YOUTUBE_TOKEN` | YouTube Data API key |
| `VELOCIREPO_CONFIG` | Path to config file |

These can also be set in a `.env` file in the current directory.

## Usage

```
velocirepo fetch                 Fetch from all configured sources
velocirepo fetch-github          Fetch GitHub events (stars, forks, issues, PRs)
velocirepo fetch-traffic         Fetch GitHub traffic (views and clones)
velocirepo fetch-pypi            Fetch PyPI download counts
velocirepo fetch-cran            Fetch CRAN download counts
velocirepo fetch-homebrew        Fetch Homebrew install counts
velocirepo fetch-plausible       Fetch Plausible analytics
velocirepo fetch-openvsx         Fetch Open VSX extension metrics
velocirepo fetch-youtube         Fetch YouTube metrics

velocirepo query <sql>           Run a SQL query against the metrics data
velocirepo schema                Show table schemas
velocirepo show-indicators       Print built-in indicator definitions as TOML

velocirepo export <dir>          Export data to Parquet or CSV files

velocirepo badge stars           Generate a stars badge
velocirepo badge forks           Generate a forks badge
velocirepo badge downloads       Generate a downloads badge
velocirepo badge pageviews       Generate a pageviews badge
velocirepo badge custom          Generate a badge from a custom query

velocirepo init                  Create a new velocirepo.toml
velocirepo add-project           Add a project to the config
velocirepo update-project        Update a project's configuration
velocirepo remove-project        Remove a project from the config
velocirepo rename-project        Rename a project's ID
velocirepo list-projects         List configured projects
velocirepo show-project          Show project details
velocirepo import-projects       Bulk-import from GitHub org/user or file
velocirepo validate-projects     Validate source URLs

velocirepo add-view <name>       Scaffold a new view directory
velocirepo remove-view <name>    Remove a view directory
velocirepo list-views            List all views
velocirepo show-view <name>      Show details about a view
velocirepo render-view <name>    Render a view (runs render.sh)
velocirepo render-views          Render all or filtered views
velocirepo serve-view <name>     Start a dev server (runs serve.sh)
velocirepo setup-views           Install dependencies for all views

velocirepo build-db              Build the DuckDB database file for external tools
velocirepo migrate               Migrate data to the latest schema version

velocirepo install-ci            Generate a GitHub Actions workflow
velocirepo sync-secrets          Sync .env secrets to GitHub Actions

velocirepo version               Print version information
```

## Data storage

All data is stored as JSONL files at `velocirepo/data/<source>/<project-id>/<date>.jsonl`. This entire directory is meant to be committed to git — it's your permanent metric history.

### Metrics sources

Sources like PyPI, CRAN, Homebrew, Plausible, OpenVSX, and GitHub Traffic store one JSON object per metric per day:

```json
{"source":"pypi","metric":"daily_downloads","project_id":"plotnine","target":"plotnine","date":"2026-06-15","value":1523}
{"source":"openvsx","metric":"total_downloads","project_id":"quarto","target":"quarto/quarto","date":"2026-06-15","value":1250000}
```

Metric names are prefixed with `daily_` (for deltas — values that reset each day) or `total_` (for snapshots — cumulative totals at a point in time). Homebrew metrics use their own naming (`downloads_30d`, `downloads_365d`, etc.).

Fields:

| Field | Description |
|-------|-------------|
| `source` | Source name (pypi, cran, homebrew, plausible, openvsx, github-traffic) |
| `metric` | Metric name (e.g., `daily_downloads`, `total_views`, `daily_pageviews`) |
| `project_id` | Project ID from your config |
| `target` | Specific package, repo, site, or extension being tracked |
| `date` | Date of the measurement (YYYY-MM-DD) |
| `value` | Integer value |
| `tags` | Optional key-value metadata (e.g., `{"video_id": "..."}` for YouTube) |

### GitHub Events source

The GitHub Events source stores individual events rather than pre-computed counts, giving you full historical detail including who performed each action and when:

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
| `datetime` | Full timestamp of the event (ISO 8601) |
| `user` | GitHub username who performed the action |

These events are automatically aggregated into daily counts in the `metrics` DuckDB view (as `daily_stars`, `daily_forks`, etc.) so you can query them alongside other sources.

### YouTube index

The YouTube source also writes an `index.jsonl` file at `velocirepo/data/youtube/<project-id>/index.jsonl` containing video metadata:

```json
{"video_id":"ML3q7Ok4hJg","title":"God-Tier Developer Roadmap","published_at":"2024-03-15T16:00:00Z","channel":"@Fireship","duration":423,"tags":["programming","roadmap"]}
```

This is exposed as the `youtube_index` DuckDB view, allowing you to join video titles and publish dates with metrics data.

### Concatenation

Daily JSONL files are automatically concatenated into monthly and yearly files once a period is complete. For example, once all days in January 2026 have been fetched, they're concatenated into `2026-01.jsonl`. This keeps the file count manageable for long-running histories. The original daily files are removed after concatenation.

### Repository layout

You can either keep metrics in the same repository as your code, or create a dedicated metrics repository. A separate repo is useful when you want to track multiple projects in one place or keep metric history out of your main codebase.

## Migrating data

When a new version of velocirepo changes the on-disk data format, it tracks this with a schema version number in `velocirepo/data/.schema-version`. Commands like `fetch`, `query`, and `export` will refuse to run against stale data:

```
Error: data schema is at version 0, but version 1 is required; run `velocirepo migrate` to update
```

To migrate your data to the latest schema:

```bash
velocirepo migrate
```

If you've copied in data from an older schema (e.g., merged data from another repository), some files may be at a different version than what `.schema-version` claims. Use `--force` to re-run all migrations from scratch:

```bash
velocirepo migrate --force
```

This is safe to run repeatedly — all migrations are idempotent.

## Querying the data

The `query` command reads JSONL files directly using DuckDB and exposes five views:

| View | Description |
|------|-------------|
| `metrics` | Unified time-series: all sources including aggregated GitHub events |
| `indicators` | Derived growth rate and trend for daily metrics (28-day windows) |
| `github_events` | Raw GitHub events with user and timestamp |
| `youtube_index` | Video metadata (title, publish date, channel, duration) |
| `projects` | Project metadata from your config |

### Daily star counts

```bash
velocirepo query "
  SELECT project, date, value AS stars
  FROM metrics
  WHERE source = 'github' AND metric = 'daily_stars'
  ORDER BY date DESC
  LIMIT 5
"
```

```
┌──────────────────┬────────────┬───────┐
│     project      │    date    │ stars │
├──────────────────┼────────────┼───────┤
│ ggsql            │ 2026-06-22 │ 1     │
│ gt               │ 2026-06-22 │ 1     │
│ images-workbench │ 2026-06-22 │ 1     │
│ rmarkdown        │ 2026-06-22 │ 1     │
│ pointblank       │ 2026-06-22 │ 1     │
└──────────────────┴────────────┴───────┘
```

### Total stars per project

```bash
velocirepo query "
  SELECT p.name, SUM(value) AS stars
  FROM metrics m
  JOIN projects p ON m.project = p.id
  WHERE m.source = 'github' AND m.metric = 'daily_stars'
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
│ cheatsheets │ 6360  │
│ shiny       │ 5655  │
│ Shiny for R │ 5600  │
│ Quarto      │ 5274  │
└─────────────┴───────┘
```

### Monthly star activity using raw events

The `github_events` view gives you access to individual events when you need per-user or per-timestamp detail:

```bash
velocirepo query "
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

### Top YouTube videos by views

```bash
velocirepo query "
  SELECT yi.title, m.value AS views
  FROM metrics m
  JOIN youtube_index yi ON m.tags->>'video_id' = yi.video_id
  WHERE m.source = 'youtube' AND m.metric = 'total_views'
  ORDER BY m.value DESC
  LIMIT 5
"
```

### Latest metrics across sources

```bash
velocirepo query "
  SELECT project, source, metric, date, value
  FROM metrics
  WHERE source != 'github'
  ORDER BY date DESC
  LIMIT 5
"
```

```
┌───────────────────┬─────────┬─────────────────┬────────────┬─────────┐
│      project      │ source  │     metric      │    date    │  value  │
├───────────────────┼─────────┼─────────────────┼────────────┼─────────┤
│ shiny-r           │ openvsx │ total_downloads │ 2026-06-22 │ 2404755 │
│ shiny-r           │ openvsx │ rating          │ 2026-06-22 │ 5       │
│ shiny-r           │ openvsx │ reviews         │ 2026-06-22 │ 2       │
│ pointblank-python │ pypi    │ daily_downloads │ 2026-06-22 │ 604     │
│ orbital-r         │ cran    │ daily_downloads │ 2026-06-22 │ 37      │
└───────────────────┴─────────┴─────────────────┴────────────┴─────────┘
```

### Output formats

By default, results are printed as a table. Use `--json`, `--csv`, or `--parquet` for machine-readable output:

```bash
velocirepo query --csv "SELECT project, metric, value FROM metrics LIMIT 3"
velocirepo query --json "SELECT project, metric, value FROM metrics LIMIT 3"
velocirepo query --parquet "SELECT * FROM metrics" > metrics.parquet
```

The `schema` command shows all available columns:

```bash
velocirepo schema
```

```
┌───────────────┬──────────────┬───────────┐
│ TABLE         │ COLUMN       │ TYPE      │
├───────────────┼──────────────┼───────────┤
│ github_events │ project      │ VARCHAR   │
│ github_events │ source       │ VARCHAR   │
│ github_events │ event_type   │ VARCHAR   │
│ github_events │ github_repo  │ VARCHAR   │
│ github_events │ datetime     │ TIMESTAMP │
│ github_events │ user         │ VARCHAR   │
│ indicators    │ project      │ VARCHAR   │
│ indicators    │ source       │ VARCHAR   │
│ indicators    │ target       │ VARCHAR   │
│ indicators    │ metric       │ VARCHAR   │
│ indicators    │ indicator    │ VARCHAR   │
│ indicators    │ date         │ DATE      │
│ indicators    │ value        │ DOUBLE    │
│ indicators    │ tags         │ JSON      │
│ metrics       │ project      │ VARCHAR   │
│ metrics       │ source       │ VARCHAR   │
│ metrics       │ target       │ VARCHAR   │
│ metrics       │ metric       │ VARCHAR   │
│ metrics       │ date         │ DATE      │
│ metrics       │ value        │ BIGINT    │
│ metrics       │ tags         │ JSON      │
│ projects      │ id           │ VARCHAR   │
│ projects      │ name         │ VARCHAR   │
│ projects      │ description  │ VARCHAR   │
│ projects      │ color        │ VARCHAR   │
│ projects      │ tags         │ VARCHAR[] │
│ projects      │ website      │ VARCHAR   │
│ projects      │ logo         │ VARCHAR   │
│ youtube_index │ video_id     │ VARCHAR   │
│ youtube_index │ title        │ VARCHAR   │
│ youtube_index │ published_at │ TIMESTAMP │
│ youtube_index │ channel      │ VARCHAR   │
│ youtube_index │ duration     │ BIGINT    │
│ youtube_index │ tags         │ JSON      │
└───────────────┴──────────────┴───────────┘
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
  out/youtube_index.parquet (15 KB)
  out/indicators.parquet (42 KB)
  out/projects.parquet (2 KB)
```

Use `--format csv` for CSV output, and `--source` or `--project` to filter:

```bash
velocirepo export ./out/ --format csv
velocirepo export ./out/ --source github
velocirepo export ./out/ --project quarto
```

## Using the data with other tools

velocirepo provides two ways to access your metrics data from external tools:

1. **DuckDB file** — a persistent `velocirepo/data/velocirepo.duckdb` file with views over the raw JSONL data and a `projects` table. Open it directly from any tool that supports DuckDB.
2. **Parquet export** — use `velocirepo export ./out/` to write Parquet (or CSV) files that any data tool can read.

The DuckDB file is rebuilt automatically after every `fetch` and project mutation command. You can also rebuild it manually:

```bash
velocirepo build-db
```

> **Note:** The DuckDB file uses relative paths, so external tools must open it from the data directory (or set DuckDB's working directory to it).

### Tool compatibility

| Tool / Package | DuckDB file | Parquet |
|---|---|---|
| DuckDB CLI | Yes | Yes |
| Python `duckdb` | Yes | Yes |
| Polars (Python/Rust) | No | Yes |
| pandas | No | Yes |
| Marimo | Yes (via `duckdb`) | Yes |
| R `duckdb` / `DBI` | Yes | Yes |
| R `arrow` | No | Yes |
| Observable / Evidence | Yes | Yes |
| Excel / Google Sheets | No | Yes (via CSV) |

### Examples

**DuckDB CLI:**

```bash
cd velocirepo/data
duckdb velocirepo.duckdb "SELECT project, SUM(value) AS stars FROM metrics WHERE metric = 'daily_stars' GROUP BY project ORDER BY stars DESC"
```

**Python (`duckdb`):**

```python
import duckdb
import os

os.chdir("velocirepo/data")
con = duckdb.connect("velocirepo.duckdb", read_only=True)
df = con.sql("SELECT * FROM metrics WHERE source = 'pypi' ORDER BY date DESC LIMIT 10").df()
```

**Polars (from Parquet export):**

```python
import polars as pl

metrics = pl.read_parquet("out/metrics.parquet")
stars = metrics.filter(pl.col("metric") == "daily_stars").group_by("project").agg(pl.col("value").sum())
```

## Indicators

The `indicators` view computes derived signals from your daily metrics, giving you a sense of how fast a project is growing and in which direction it's heading. Indicators are computed using 28-day trailing windows and are available wherever you query data — via `velocirepo query`, the persistent `.duckdb` file, and Parquet exports.

Only metrics with a `daily_` prefix are included (these represent per-day deltas like `daily_stars`, `daily_downloads`, `daily_pageviews`). Cumulative `total_*` metrics are excluded since growth rates on snapshots aren't meaningful.

### Schema

| Column | Type | Description |
|--------|------|-------------|
| `project` | VARCHAR | Project ID |
| `source` | VARCHAR | Source name (github, pypi, etc.) |
| `target` | VARCHAR | Specific package, repo, or site being tracked |
| `metric` | VARCHAR | Underlying metric (e.g., `daily_stars`) |
| `indicator` | VARCHAR | Indicator name (`growth_rate` or `trend`) |
| `date` | DATE | Date of computation |
| `value` | DOUBLE | Computed value |
| `tags` | JSON | Optional metadata from the underlying metric |

### Growth rate

Measures how much activity increased or decreased compared to the prior period. Computed as:

```
growth_rate = (sum_last_28d - sum_prior_28d) / sum_prior_28d
```

A value of `0.15` means 15% more activity in the last 28 days compared to the 28 days before that. Negative values indicate declining activity.

```bash
velocirepo query "
  SELECT project, metric, date, ROUND(value, 3) AS growth_rate
  FROM indicators
  WHERE indicator = 'growth_rate'
    AND metric = 'daily_stars'
  ORDER BY date DESC
  LIMIT 5
"
```

```
┌──────────────┬─────────────┬────────────┬─────────────┐
│   project    │   metric    │    date    │ growth_rate │
├──────────────┼─────────────┼────────────┼─────────────┤
│ py-shiny     │ daily_stars │ 2026-06-22 │ 0.114       │
│ dplyr        │ daily_stars │ 2026-06-22 │ -0.182      │
│ plotnine     │ daily_stars │ 2026-06-22 │ -0.488      │
│ great-tables │ daily_stars │ 2026-06-22 │ -0.174      │
│ ggsql        │ daily_stars │ 2026-06-22 │ -0.875      │
└──────────────┴─────────────┴────────────┴─────────────┘
```

### Trend

Measures the daily rate of change via linear regression over the trailing 28 days. The value represents units per day — for example, a trend of `3.2` on `daily_stars` means the project is gaining roughly 3.2 more stars per day than it was at the start of the window.

```bash
velocirepo query "
  SELECT project, metric, date, ROUND(value, 2) AS trend_per_day
  FROM indicators
  WHERE indicator = 'trend'
    AND metric = 'daily_downloads'
    AND project = 'plotnine'
  ORDER BY date DESC
  LIMIT 5
"
```

```
┌──────────┬─────────────────┬────────────┬───────────────┐
│ project  │     metric      │    date    │ trend_per_day │
├──────────┼─────────────────┼────────────┼───────────────┤
│ plotnine │ daily_downloads │ 2026-06-22 │ -140.39       │
│ plotnine │ daily_downloads │ 2026-06-21 │ -60.35        │
│ plotnine │ daily_downloads │ 2026-06-20 │ 135.05        │
│ plotnine │ daily_downloads │ 2026-06-19 │ 186.6         │
│ plotnine │ daily_downloads │ 2026-06-18 │ 112.38        │
└──────────┴─────────────────┴────────────┴───────────────┘
```

### Querying indicators

You can join indicators with project metadata for richer views:

```bash
velocirepo query "
  SELECT p.name, i.metric, i.indicator, i.date, ROUND(i.value, 3) AS value
  FROM indicators i
  JOIN projects p ON i.project = p.id
  WHERE i.date = (SELECT MAX(date) FROM indicators)
  ORDER BY i.indicator, i.value DESC
  LIMIT 5
"
```

```
┌────────────┬─────────────────────┬─────────────┬────────────┬───────┐
│    name    │       metric        │  indicator  │    date    │ value │
├────────────┼─────────────────────┼─────────────┼────────────┼───────┤
│ great-docs │ daily_issues_closed │ growth_rate │ 2026-06-22 │ 5.143 │
│ mcp-repl   │ daily_prs_merged    │ growth_rate │ 2026-06-22 │ 3.143 │
│ mcp-repl   │ daily_prs_opened    │ growth_rate │ 2026-06-22 │ 2.815 │
│ great-docs │ daily_issues_opened │ growth_rate │ 2026-06-22 │ 1.647 │
│ Positron   │ daily_pr_comment    │ growth_rate │ 2026-06-22 │ 1.271 │
└────────────┴─────────────────────┴─────────────┴────────────┴───────┘
```

### Custom indicators

You can define your own indicators in `velocirepo.toml`. Each indicator is a named TOML table with a description and a SQL query:

```toml
[indicators.weekly_momentum]
description = "7-day trailing sum"
query = """
  SELECT project, source, target, metric,
    '{{indicator_name}}' AS indicator, date,
    SUM(value) OVER w AS value,
    tags
  FROM metrics WHERE metric LIKE 'daily_%'
  WINDOW w AS (PARTITION BY project, source, target, metric, tags ORDER BY date ROWS 6 PRECEDING)
"""
```

The `{{indicator_name}}` placeholder is replaced with the TOML key at runtime (e.g., `weekly_momentum`).

**Behavior:** Custom indicators are added alongside the built-in defaults. To disable the defaults, set:

```toml
[settings]
include_default_indicators = false
```

To inspect the built-in defaults:

```bash
velocirepo show-indicators --defaults
```

Without `--defaults`, `show-indicators` prints all active indicators (defaults + custom). You can redirect to your config to use as a starting point:

```bash
velocirepo show-indicators --defaults >> velocirepo.toml
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

## Views

Views are self-contained directories that produce dashboards, reports, or visualizations from your metrics data. Each view has a `render.sh` script as the contract — velocirepo handles discovery and orchestration, while rendering is delegated to whatever tools the view needs.

### Structure

A view is any directory under `views/` that contains a `render.sh` file:

```
views/
  weekly-stars/
    render.sh          # the render contract (executable)
    pyproject.toml     # Python dependencies (managed by uv)
    view.qmd           # the actual view file
  monthly-report/
    render.sh
    pyproject.toml
    app.py             # marimo notebook
  r-summary/
    render.sh
    view.R
```

### Quick start

```bash
# Scaffold a Quarto view (creates directory with render.sh, view.qmd, pyproject.toml)
velocirepo add-view weekly-stars --framework quarto

# Install dependencies
velocirepo setup-views

# Render the view (rebuilds DuckDB, then runs render.sh)
velocirepo render-view weekly-stars

# Or start a live dev server (requires serve.sh in the view)
velocirepo serve-view weekly-stars
```

### Supported frameworks

| Framework | Flag | View file | render.sh runs |
|-----------|------|-----------|----------------|
| [Quarto](https://quarto.org) | `-f quarto` | `view.qmd` | `uv run quarto render view.qmd` |
| [Jupyter](https://jupyter.org) | `-f jupyter` | `view.ipynb` | `uv run jupyter nbconvert ...` |
| [Marimo](https://marimo.io) | `-f marimo` | `app.py` | `uv run marimo export html app.py` |
| R | `-f r` | `view.R` | `Rscript view.R` |
| [ggsql](https://ggsql.io) | `-f sql` | `view.sql` | `ggsql run view.sql` |

### Data access

Views connect to `velocirepo.duckdb` by default (via a relative path embedded at scaffold time). The DB is rebuilt before every render, so views always see fresh data.

For views that need Parquet files instead, use `--source parquet` when scaffolding and call `velocirepo export` in the view's own `render.sh`.

### Flags for `add-view`

| Flag | Description |
|------|-------------|
| `-f, --framework` | Framework: quarto, jupyter, marimo, r, sql (required) |
| `-s, --source` | Data source: duckdb (default) or parquet |
| `--no-uv` | Skip pyproject.toml generation |
| `--renv` | Scaffold renv for R views |

### Rendering

```bash
velocirepo render-view weekly-stars  # render one view
velocirepo render-views              # render all views
velocirepo render-views reports      # render views matching a prefix
```

### Setup and CI

```bash
# Install dependencies for all views (runs uv sync / renv::restore)
velocirepo setup-views
```

In GitHub Actions:

```yaml
- name: Setup views
  run: velocirepo setup-views

- name: Render views
  run: velocirepo render-views
```

## MCP server

velocirepo includes a built-in [Model Context Protocol](https://modelcontextprotocol.io) (MCP) server, allowing AI assistants like Claude to query your metrics, trigger fetches, and manage projects conversationally.

### Starting the server

```bash
velocirepo mcp                          # stdio (for Claude Desktop / Claude Code)
velocirepo mcp --http 127.0.0.1:8080    # Streamable HTTP
velocirepo mcp --read-only              # Disable fetch/write tools
```

### Claude Desktop configuration

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "velocirepo": {
      "command": "velocirepo",
      "args": ["mcp", "--config", "/path/to/velocirepo.toml"]
    }
  }
}
```

### Claude Code configuration

Add to your project's `.mcp.json`:

```json
{
  "mcpServers": {
    "velocirepo": {
      "command": "velocirepo",
      "args": ["mcp"]
    }
  }
}
```

### Available tools

| Tool | Description |
|------|-------------|
| `query` | Run SQL against metrics, github_events, youtube_index, projects views |
| `schema` | Show all table columns and types |
| `fetch` | Fetch from all configured sources |
| `fetch_github` | Fetch GitHub events |
| `fetch_traffic` | Fetch GitHub traffic |
| `fetch_pypi` | Fetch PyPI downloads |
| `fetch_cran` | Fetch CRAN downloads |
| `fetch_homebrew` | Fetch Homebrew installs |
| `fetch_plausible` | Fetch Plausible analytics |
| `fetch_openvsx` | Fetch Open VSX metrics |
| `fetch_youtube` | Fetch YouTube metrics |
| `list_projects` | List configured projects |
| `show_project` | Show project details and fetch stats |
| `add_project` | Add a project to the config |
| `update_project` | Update project configuration |
| `remove_project` | Remove a project |
| `rename_project` | Rename a project ID |
| `import_projects` | Bulk-import from GitHub org/user |
| `validate_projects` | Check that source URLs are reachable |
| `list_views` | List all views with framework and source |
| `show_view` | Show view details |
| `add_view` | Scaffold a new view |
| `remove_view` | Remove a view |
| `render_view` | Render a single view |
| `render_views` | Render all or filtered views |
| `badge` | Generate an SVG badge |
| `export` | Export data to Parquet or CSV |
| `migrate` | Migrate data to latest schema |
| `version` | Show version info |

### Example prompts

Once connected, you can ask things like:

- "Which project got the most stars this month?"
- "Show me PyPI download trends for plotnine over the last 6 months"
- "Fetch the latest metrics for all projects"
- "Add a new project called 'my-lib' tracking pypi package my-lib and github owner/my-lib"
- "Generate a stars badge for quarto"

## License

MIT
