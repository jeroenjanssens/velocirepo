# velocirepo

Fetch and aggregate open-source project metrics into a queryable, git-friendly history.

## Table of contents

- [Overview](#overview)
- [Installation](#installation)
- [Configuration](#configuration)
- [Usage](#usage)
- [Data storage](#data-storage)
- [Migrating data](#migrating-data)
- [Querying the data](#querying-the-data)
- [Exporting data](#exporting-data)
- [Generating badges](#generating-badges)

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

## Configuration

Create a `velocirepo.toml` in your project root:

```toml
[projects.my-project]
name = "My Project"
github-events = "owner/repo"
github-traffic = "owner/repo"
pypi = "my-package"

[projects.other-project]
name = "Other Project"
github-events = ["owner/other", "owner/other-utils"]
cran = "other"
homebrew = "other"
youtube = "@ChannelHandle"
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
| `PLAUSIBLE_TOKEN` | Plausible API token |
| `YOUTUBE_TOKEN` | YouTube Data API key |
| `VELOCIREPO_CONFIG` | Path to config file |

These can also be set in a `.env` file in the current directory.

## Usage

```
velocirepo fetch github-events   Fetch GitHub events (stars, forks, issues, PRs)
velocirepo fetch github-traffic  Fetch GitHub traffic (views and clones)
velocirepo fetch pypi            Fetch PyPI download counts
velocirepo fetch cran            Fetch CRAN download counts
velocirepo fetch homebrew        Fetch Homebrew install counts
velocirepo fetch plausible       Fetch Plausible analytics
velocirepo fetch openvsx         Fetch Open VSX extension metrics
velocirepo fetch youtube         Fetch YouTube metrics
velocirepo fetch all             Fetch from all configured sources

velocirepo query run [sql]       Run a SQL query against the metrics data
velocirepo query schema          Show table schemas

velocirepo export <dir>          Export data to Parquet or CSV files

velocirepo badge stars           Generate a stars badge
velocirepo badge forks           Generate a forks badge
velocirepo badge downloads       Generate a downloads badge
velocirepo badge pageviews       Generate a pageviews badge
velocirepo badge custom          Generate a badge from a custom query

velocirepo project init          Create a new velocirepo.toml
velocirepo project add           Add a project to the config
velocirepo project update        Update a project's configuration
velocirepo project remove        Remove a project from the config
velocirepo project rename        Rename a project's ID
velocirepo project list          List configured projects
velocirepo project show          Show project details
velocirepo project import        Bulk-import from GitHub org/user or file
velocirepo project validate      Validate source URLs

velocirepo migrate               Migrate data to the latest schema version

velocirepo ci install            Generate a GitHub Actions workflow

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

The GitHub Events source stores individual events rather than aggregated counts, giving you full historical detail including who performed each action and when:

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

These events are automatically aggregated into daily counts in the `metrics` DuckDB view (as `daily_star`, `daily_fork`, etc.) so you can query them alongside other sources.

### YouTube index

The YouTube source also writes an `index.jsonl` file at `velocirepo/data/youtube/<project-id>/index.jsonl` containing video metadata:

```json
{"video_id":"ML3q7Ok4hJg","title":"God-Tier Developer Roadmap","published_at":"2024-03-15T16:00:00Z","channel":"@Fireship","duration":423,"tags":["programming","roadmap"]}
```

This is exposed as the `youtube_index` DuckDB view, allowing you to join video titles and publish dates with metrics data.

### Aggregation

Daily JSONL files are automatically rolled up into monthly and yearly files once a period is complete. For example, once all days in January 2026 have been fetched, they're aggregated into `2026-01.jsonl`. This keeps the file count manageable for long-running histories. The original daily files are removed after aggregation.

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

The `query` command reads JSONL files directly using DuckDB and exposes four views:

| View | Description |
|------|-------------|
| `metrics` | Unified time-series: all sources including aggregated GitHub events |
| `github_events` | Raw GitHub events with user and timestamp |
| `youtube_index` | Video metadata (title, publish date, channel, duration) |
| `projects` | Project metadata from your config |

### Daily star counts

```bash
velocirepo query run "
  SELECT project, date, value AS stars
  FROM metrics
  WHERE source = 'github' AND metric = 'daily_star'
  ORDER BY date DESC
  LIMIT 5
"
```

### Total stars per project

```bash
velocirepo query run "
  SELECT p.name, SUM(value) AS stars
  FROM metrics m
  JOIN projects p ON m.project = p.id
  WHERE m.source = 'github' AND m.metric = 'daily_star'
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

### Monthly star activity using raw events

The `github_events` view gives you access to individual events when you need per-user or per-timestamp detail:

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

### Top YouTube videos by views

```bash
velocirepo query run "
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
velocirepo query run "
  SELECT project, source, metric, date, value
  FROM metrics
  WHERE source != 'github'
  ORDER BY date DESC
  LIMIT 5
"
```

```
┌──────────┬─────────┬─────────────────┬────────────┬─────────┐
│ project  │ source  │     metric      │    date    │  value  │
├──────────┼─────────┼─────────────────┼────────────┼─────────┤
│ quarto   │ openvsx │ total_downloads │ 2026-06-16 │ 3101234 │
│ quarto   │ openvsx │ total_rating    │ 2026-06-16 │ 500     │
│ quarto   │ openvsx │ total_reviews   │ 2026-06-16 │ 2       │
│ quarto   │ plausible│ daily_pageviews │ 2026-06-16 │ 14322   │
│ quarto   │ plausible│ daily_visitors  │ 2026-06-16 │ 3324    │
└──────────┴─────────┴─────────────────┴────────────┴─────────┘
```

### Output formats

By default, results are printed as a table. Use `--json`, `--csv`, or `--parquet` for machine-readable output:

```bash
velocirepo query run --csv "SELECT project, metric, value FROM metrics LIMIT 3"
velocirepo query run --json "SELECT project, metric, value FROM metrics LIMIT 3"
velocirepo query run --parquet "SELECT * FROM metrics" > metrics.parquet
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
  out/youtube_index.parquet (15 KB)
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

## License

MIT
