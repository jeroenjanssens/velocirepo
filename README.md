# velocirepo

A CLI tool for fetching and aggregating open-source project metrics from GitHub, PyPI, CRAN, Plausible, and OpenVSX.

## Installation

### Homebrew (macOS / Linux)

```bash
brew install jeroenjanssens/tap/velocirepo
```

### Scoop (Windows)

```powershell
scoop bucket add jeroenjanssens https://github.com/jeroenjanssens/scoop-bucket
scoop install velocirepo
```

### Go

```bash
go install github.com/jeroenjanssens/velocirepo/cmd/velocirepo@latest
```

### From source

```bash
git clone https://github.com/jeroenjanssens/velocirepo.git
cd velocirepo
go build -o bin/velocirepo ./cmd/velocirepo
cp bin/velocirepo ~/.local/bin/
```

### Download binary

Pre-built binaries for Linux, macOS, and Windows are available on the [Releases](https://github.com/jeroenjanssens/velocirepo/releases) page.

## Quick start

Create a `velocirepo.toml` in your project:

```toml
[data]
dir = "data"

[project]
name = "My Project"
github = "owner/repo"
pypi = "my-package"
```

Fetch metrics:

```bash
velocirepo fetch all
```

Query the data:

```bash
velocirepo query run "SELECT metric, date, value FROM metrics ORDER BY date DESC LIMIT 10"
velocirepo query schema
```

## Usage

```
velocirepo fetch github      Fetch GitHub metrics (stars, forks, issues, PRs)
velocirepo fetch pypi        Fetch PyPI download counts
velocirepo fetch cran        Fetch CRAN download counts
velocirepo fetch plausible   Fetch Plausible analytics
velocirepo fetch openvsx     Fetch Open VSX install counts
velocirepo fetch all         Fetch from all configured sources

velocirepo query run [sql]   Run a SQL query against the metrics data
velocirepo query schema      Show the metrics table schema

velocirepo ci install        Generate a GitHub Actions workflow

velocirepo version           Print version information
```

## Configuration

velocirepo looks for `velocirepo.toml` by walking up from the current directory. Override with `--config` or the `VELOCIREPO_CONFIG` environment variable.

### Single project

```toml
[data]
dir = "data"

[project]
name = "My Project"
github = "owner/repo"
pypi = "my-package"
plausible = "example.com"
openvsx = "publisher/extension"
```

### Multiple projects

```toml
[data]
dir = "data"

[projects.my-project]
name = "My Project"
github = "owner/repo"
pypi = "my-package"

[projects.other-project]
name = "Other Project"
github = "owner/other"
cran = "other"
```

## Environment variables

| Variable | Description |
|----------|-------------|
| `GITHUB_TOKEN` | GitHub personal access token (increases rate limits) |
| `PLAUSIBLE_KEY` | Plausible API key |
| `VELOCIREPO_CONFIG` | Path to config file |

## Data storage

Metrics are stored as JSONL files at `data/<source>/<project-id>/<date>.jsonl`. These files are designed to be committed to git so you have a complete history of your project metrics.

The `query` command reads JSONL files directly using DuckDB — no build step required.

## License

MIT
