# AGENTS.md

## Purpose

`sbot` — Go service that auto-generates reports from data sources (Metabase, Jira, AppMetrica, Zabbix), processes them through a pipeline (Lua scripts or DuckDB SQL), renders templates, and delivers results to Telegram, Max Messenger, SMTP, and SMB shares. Report schedules live in PostgreSQL (`crons` table). Also provides a Telegram bot UI (admin/user roles) and a management CLI (`sbot ctl ...`). README and domain docs are in Russian.

## Commands

Build/run is driven by **Task** (`Taskfile.yml`), not Make — README mentions `make`, but no Makefile exists.

```bash
task build        # static linux/amd64 binary -> bin/sbot (CGO via musl-gcc, -tags chromium, -mod=mod)
task run          # go run with config/config/local.yaml (override: task run CONFIG_NAME=other.yaml)
task docker-run   # build static binary inside Docker and copy to ./bin/sbot
go test ./...     # no task defined; run tests directly
golangci-lint run # config in .golangci.yml (pinned to v2.6.0)
```

- Module name is `support_bot` (imports look like `support_bot/internal/...`); Go 1.26.
- Dependencies are vendored (`vendor/`), but builds use `-mod=mod`. After adding a dep, run `go mod tidy && go mod vendor`.
- PDF backend is selected by build tags: `-tags chromium` → headless Chromium, `-tags wkhtmltopdf` → wkhtmltopdf, no tag → stub that errors (`internal/exporter/pdf/default.go`). CI/Taskfile uses `chromium`.

## Layout

- `cmd/bot` — main `sbot` binary; `cmd/live-server` — live-preview server for template editing.
- `internal/collector` — data sources; `internal/processor` — pipeline (Lua, DuckDB SQL); `internal/generator` + `internal/orchestrator` — report generation flow; `internal/sheduler` (sic — misspelled name is load-bearing, keep it) — cron scheduler; `internal/event_creator` — cron→report linking.
- `internal/exporter` — output formats (csv, text, html, png, xlsx, pdf) behind `exporter.Exporter` interface, registered in `exporter.go`.
- `internal/delivery` — Telegram, Max, SMTP, SMB senders; `internal/tg_bot`, `internal/max_bot` — bot UIs.
- `internal/cli` — CLI commands (`ctl export/apply/validate`, `config`, `script`); `internal/config` — config loading (`--config` flag → `CONFIG_PATH` env → `./config.yaml` → env/`.env`).
- `internal/models` — shared domain types; `internal/pkg` — logger, template funcs, helpers; `internal/errorz` — sentinel errors.
- `migrations/` — DB migrations; `lua/`, `private_plugins/` — Lua scripts; `config/local*.yaml`, `.env`, `reports/` are local-only secrets/artifacts — never commit them.

## Conventions

- Logging goes through the `log/slog` wrapper in `internal/pkg/logger`, not stdlib `log` or fmt-based logging.
- Lint is strict: gofumpt/goimports/golines formatters; errcheck, errorlint, funcorder (constructors before other methods, exported before unexported), errname, gosec, gocritic with all tags. Run `golangci-lint run` before finishing Go changes.
- Report data flows as `models.Dataset` (`map[string][]map[string]any`, keyed by query `title`); template functions live in `internal/pkg/text/func_map.go` (Sprig + custom) and `internal/pkg/funcs`.

## Current work (branch `exportV2`)

In-progress HTML/layout export system: `internal/exporter/html/{layout,styles,table,layout_funcs}.go`, `internal/models/export_layout_template.go`, `internal/models/layout.json`, plus Chart.js template functions (`lineChart`/`barChart`/`pieChart`) in `internal/exporter/html/chart.go`. Read `TODO.md` and `internal/exporter/html/task.md` before touching the HTML/PDF exporters — they contain the spec.
