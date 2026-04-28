package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/vstratful/sanity-cli/internal/config"
)

var agentSetupCmd = &cobra.Command{
	Use:   "agent-setup",
	Short: "Print a structured setup guide for AI agents",
	Long: `Print context and setup instructions for an AI coding agent.

This command does NOT require any configuration and can be run immediately
after installation. It outputs the config file location, environment variable
names, command list, and JSON output conventions so an agent can self-configure.`,
	Run: runAgentSetup,
}

func init() {
	rootCmd.AddCommand(agentSetupCmd)
}

func runAgentSetup(cmd *cobra.Command, args []string) {
	resolvedPath, err := config.GetConfigPath()
	if err != nil {
		resolvedPath = "(unable to determine)"
	}
	desc := configPathDescription()

	fmt.Printf(`# sanity-cli — Agent Setup Guide

## Configuration

Config file location: %s
(Resolved on this machine: %s)

A typical config.json looks like:

{
  "current_instance": "production",
  "default_project_id": "abc123",
  "instances": {
    "production": {
      "project_id": "abc123",
      "dataset": "production",
      "token": "skXXX...",
      "api_version": "%s",
      "use_cdn": false,
      "perspective": "%s"
    }
  }
}

## Environment variables

  SANITY_PROJECT_ID, SANITY_DATASET, SANITY_TOKEN
    When all three are set, sanity-cli uses an ephemeral instance built from
    the environment and ignores config.json. Useful for CI.

  SANITY_API_VERSION, SANITY_USE_CDN, SANITY_PERSPECTIVE
    Optional overrides for the ephemeral env instance.

  SANITY_CLI_INSTANCE
    Selects a named instance from config.json without --instance.

  SANITY_CLI_AUTO_CONFIRM=1
    Allows ` + "`mutate`" + ` to apply mutations without --confirm. Use only in
    trusted automation.

## Commands

  # NOTE for agents: prefer flag-based commands. 'init' and the bare
  # 'instance switch' (no name) are INTERACTIVE — they read from stdin /
  # render a TUI picker and will hang or fail under automation.

  sanity-cli instance add <name> --project ... --dataset ... --token ... [--api-version ...] [--current]
  sanity-cli instance list
  sanity-cli instance switch <name>               # always pass a name from automation
  sanity-cli instance show [name]
  sanity-cli instance remove <name> --yes

  # Interactive (humans only):
  sanity-cli init                                 # prompts on stdin
  sanity-cli instance switch                      # opens a bubbletea picker

  sanity-cli query '<groq>' [--params '<json>'] [--raw]
  sanity-cli mutate [file|-] --confirm [--dry-run] [--return-ids] [--return-documents]
    # Input is a BARE JSON ARRAY of mutation objects — NOT {"mutations":[...]}.
    # (The wrapper shape is also tolerated, but the bare array is canonical.)
    # Example file contents:
    #   [
    #     {"patch":  {"id": "<docId>", "set": {"title": "Updated"}}},
    #     {"create": {"_type": "post", "title": "New post"}},
    #     {"delete": {"id": "<docId>"}}
    #   ]

  sanity-cli schema introspect [--sample-size N] [--max-depth N] [--no-cache] [--resolve-references]
  sanity-cli schema show [--refresh]

  sanity-cli asset upload <path> --type image|file [--label ...] [--title ...]

  sanity-cli project list
  sanity-cli dataset list [--project <id>]

  sanity-cli update [--check] [--force]
  sanity-cli agent-setup

## Output convention

Every command emits a single JSON document to stdout. Use --pretty for indented
output, --instance <name> to override the active instance.

  Success envelope:  {"ok": true, "data": <result>}
  Schema commands:   {"ok": true, "data": {...}, "cached_at": "...", "cache_path": "..."}
  Error envelope:    {"ok": false, "error": "<code>", "message": "...", "details": {...}}

The 'query' subcommand additionally accepts --raw to strip the envelope and
emit the bare GROQ result for direct piping into 'jq'.

## Recommended workflow (agents/automation)

The two configuration paths below are equivalent. Pick one — do not run the
interactive 'init' command.

  Option A — persist credentials in config:
    sanity-cli instance add prod \
        --project <id> --dataset <name> --token <sk...> --current
    # subsequent commands use the saved instance

  Option B — pass credentials per invocation via env (no config file needed):
    export SANITY_PROJECT_ID=<id>
    export SANITY_DATASET=<name>
    export SANITY_TOKEN=<sk...>
    # the env trio overrides config; useful in CI

Then:

  1. sanity-cli schema introspect --pretty        # learn the data model
  2. sanity-cli query '*[_type == "post"][0..2]'  # read
  3. sanity-cli mutate ./changes.json --dry-run   # preview a write
  4. sanity-cli mutate ./changes.json --confirm   # apply
`, desc, resolvedPath, config.DefaultAPIVersion, config.DefaultPerspective)
}

func configPathDescription() string {
	switch runtime.GOOS {
	case "darwin":
		return "~/Library/Application Support/sanity-cli/config.json"
	case "windows":
		return `%APPDATA%\sanity-cli\config.json`
	default:
		return "~/.config/sanity-cli/config.json"
	}
}
