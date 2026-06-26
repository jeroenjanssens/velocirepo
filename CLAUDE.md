# velocirepo

Go CLI tool for fetching and aggregating open-source project metrics. Uses a flat command structure (verb-first, no subcommands).

## Build

```
go build -o bin/velocirepo ./cmd/velocirepo
```

## Test

```
go test ./...
```

All tests run offline using httptest — no network access required.

## Project layout

- `cmd/velocirepo/` — CLI entrypoint and Cobra commands
- `internal/config/` — TOML config loading (BurntSushi/toml)
- `internal/source/` — Fetcher implementations (GitHub Events, GitHub Traffic, PyPI, CRAN, Homebrew, Plausible, OpenVSX, YouTube)
- `internal/store/` — JSONL read/write, concatenation, DuckDB build
- `internal/version/` — Version vars injected via ldflags

## Conventions

- Each source implements the `source.Source` interface
- Source fields in config accept a string or array of strings (`config.StringList`)
- Tests use `net/http/httptest` with canned responses — no mocking frameworks
- Config file is `velocirepo.toml`, discovered by walking up from cwd
- Data stored as JSONL files at `velocirepo/data/metrics/<source>/<project-id>/<date>.jsonl` and `velocirepo/data/events/<source>/<project-id>/<date>.jsonl`
- Concatenation runs automatically after fetch (daily→monthly→yearly)
