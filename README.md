# sanity-cli

Agent-friendly Go CLI for interacting with a [Sanity.io](https://www.sanity.io) project.

Supports multiple named instances (project + dataset + token + apiVersion), JSON-first output for piping into agents, GROQ queries, mutations (gated by `--confirm`), live schema introspection, asset uploads, and Manage API listing.

## Install

The binary must end up in a directory that's **already in `$PATH`** so it can
be invoked as plain `sanity-cli`. Don't drop it in the project's working
directory or `$HOME` root — agents and shells expect it on `$PATH`.

Pick the first writable directory from this list that's already in `$PATH`:

1. `~/.local/bin` (no sudo, preferred)
2. `~/bin` (no sudo)
3. `/usr/local/bin` (needs sudo)

A shell snippet that resolves the right one:

```bash
INSTALL_DIR=$(
  for d in "$HOME/.local/bin" "$HOME/bin" /usr/local/bin; do
    case ":$PATH:" in *":$d:"*)
      if [ -w "$d" ] || [ "$(id -u)" -eq 0 ]; then echo "$d"; break; fi
    esac
  done
)
echo "Will install to: $INSTALL_DIR"
```

If none of those are on `$PATH`, add `~/.local/bin` to `$PATH` first
(append `export PATH="$HOME/.local/bin:$PATH"` to `~/.bashrc` /
`~/.zshrc`) and create the directory.

### Option A — go install (if Go is available)

```bash
go install github.com/vstratful/sanity-cli@latest
# binary lands at $(go env GOPATH)/bin/sanity-cli — ensure that's on $PATH
```

### Option B — prebuilt binary from Releases

```bash
# Detect platform
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m); case "$ARCH" in x86_64) ARCH=amd64 ;; aarch64|arm64) ARCH=arm64 ;; esac

# Resolve the latest tag (vX.Y.Z) and strip the leading 'v' for the asset name
TAG=$(curl -fsSL https://api.github.com/repos/vstratful/sanity-cli/releases/latest | grep -m1 '"tag_name"' | cut -d'"' -f4)
VERSION=${TAG#v}

# Download + extract
curl -fsSL "https://github.com/vstratful/sanity-cli/releases/download/${TAG}/sanity-cli_${VERSION}_${OS}_${ARCH}.tar.gz" \
  | tar -xz -C "$INSTALL_DIR" sanity-cli
chmod +x "$INSTALL_DIR/sanity-cli"
sanity-cli --version
```

### Option C — build from source

```bash
git clone https://github.com/vstratful/sanity-cli.git
cd sanity-cli
go build -o "$INSTALL_DIR/sanity-cli"
sanity-cli --version
```

## Quick start

> **Agents and scripts:** use the flag-based path below. Do **not** run
> `sanity-cli init` — it's an interactive prompt and will hang under
> automation. Run `sanity-cli agent-setup` for a guide intended for AI
> agents.

```bash
# Option A — persist credentials in config (recommended for repeated use):
sanity-cli instance add prod \
    --project <id> --dataset <name> --token <sk...> --current

# Option B — pass credentials via env per invocation (good for CI):
export SANITY_PROJECT_ID=<id>
export SANITY_DATASET=<name>
export SANITY_TOKEN=<sk...>

# Then:
sanity-cli schema introspect --pretty             # learn the data model
sanity-cli query '*[_type == "post"][0..2]{_id,title}' --pretty
```

Humans can use `sanity-cli init` for an interactive first-run wizard instead.

## Configuration

Config file: `~/.config/sanity-cli/config.json` (Linux), `~/Library/Application Support/sanity-cli/config.json` (macOS), `%APPDATA%\sanity-cli\config.json` (Windows). File mode `0600`.

```json
{
  "current_instance": "production",
  "instances": {
    "production": {
      "project_id": "abc123",
      "dataset": "production",
      "token": "skXXX...",
      "api_version": "2024-10-01",
      "use_cdn": false,
      "perspective": "published"
    }
  }
}
```

### Environment variables

| Var | Purpose |
| --- | --- |
| `SANITY_PROJECT_ID` + `SANITY_DATASET` + `SANITY_TOKEN` | Use an ephemeral instance built from env (skips config). |
| `SANITY_API_VERSION`, `SANITY_USE_CDN`, `SANITY_PERSPECTIVE` | Optional overrides for the env-based instance. |
| `SANITY_CLI_INSTANCE` | Pick a named instance from config without `--instance`. |
| `SANITY_CLI_AUTO_CONFIRM=1` | Allow `mutate` without `--confirm`. Use only in trusted CI. |

## Commands

```text
sanity-cli instance add <name> --project ... --dataset ... --token ... [--current]
sanity-cli instance list|switch <name>|show [name]|remove <name> --yes
sanity-cli query '<groq>' [--params '<json>'] [--raw]
sanity-cli mutate [file|-] --confirm [--dry-run] [--return-ids] [--return-documents]
sanity-cli schema introspect [--sample-size N] [--max-depth N] [--no-cache] [--resolve-references]
sanity-cli schema show [--refresh]
sanity-cli asset upload <path> --type image|file
sanity-cli project list
sanity-cli dataset list [--project <id>]
sanity-cli agent-setup
sanity-cli update

# Interactive (humans only — will hang under automation):
sanity-cli init                                   # first-run wizard
sanity-cli instance switch                        # bubbletea picker (no name arg)
```

Run `sanity-cli agent-setup` for a structured guide intended for AI agents.

## Output

Every command emits JSON to stdout. Add `--pretty` for indented output.

```json
{"ok": true, "data": ...}
{"ok": false, "error": "<code>", "message": "...", "details": ...}
```

`query --raw` strips the envelope and emits the bare GROQ result for direct piping into `jq`.

## Mutation safety

`mutate` refuses to run unless `--confirm` is set or `SANITY_CLI_AUTO_CONFIRM=1` is in the environment. `--dry-run` always supersedes; the input is parsed and previewed but never sent.

## Development

```bash
go build -o sanity-cli && go vet ./... && go test ./...
```

## License

MIT — see [LICENSE](./LICENSE).
