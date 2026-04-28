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

  sanity-cli init                                 First-run wizard
  sanity-cli instance add <name> --project ... --dataset ... --token ...
  sanity-cli instance list
  sanity-cli instance switch [name]               Interactive picker if no name
  sanity-cli instance show [name]
  sanity-cli instance remove <name> --yes

  sanity-cli query '<groq>' [--params '<json>'] [--raw]
  sanity-cli mutate [file|-] --confirm [--dry-run] [--return-ids] [--return-documents]

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

## Recommended workflow

  1. sanity-cli init                              # capture credentials
  2. sanity-cli schema introspect --pretty        # learn the data model
  3. sanity-cli query '*[_type == "post"][0..2]'  # read
  4. sanity-cli mutate ./changes.json --dry-run   # preview a write
  5. sanity-cli mutate ./changes.json --confirm   # apply
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
