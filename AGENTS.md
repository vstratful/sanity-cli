# sanity-cli

Go CLI for Sanity.io. JSON-first output. Designed for AI agents and scripts.

## Development

```bash
go build -o sanity-cli && go vet ./... && go test ./...
```

## Project Structure

- `main.go` — entry point
- `cmd/` — Cobra commands (one file per subcommand; each `init()` registers itself)
- `internal/api/` — thin HTTP client (Query, Mutate, UploadAsset, ListProjects, ListDatasets) with retry/backoff
- `internal/config/` — multi-instance config at `~/.config/sanity-cli/config.json`
- `internal/schema/` — live-API schema introspection (samples GROQ, infers field shapes, caches to disk)
- `internal/tui/picker/` — Bubble Tea instance picker
- `internal/update/` — go-selfupdate wrapper

## Conventions

- Every command emits a JSON envelope to stdout (`{ok: true, data}` or `{ok: false, error, message, details}`). Errors go to stdout (still JSON-first) with non-zero exit.
- Mutations require `--confirm` flag or `SANITY_CLI_AUTO_CONFIRM=1`.
- Instance precedence: `SANITY_PROJECT_ID/DATASET/TOKEN` env trio → `--instance` flag → `SANITY_CLI_INSTANCE` env → `current_instance` in config → first-run prompt.
- `use_cdn` only flips the host for read queries; mutations / Manage API / asset uploads always hit `api.sanity.io`.
