package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"github.com/vstratful/sanity-cli/internal/api"
	"github.com/vstratful/sanity-cli/internal/schema"
)

var (
	introspectSampleSize int
	introspectMaxDepth   int
	introspectNoCache    bool
	introspectResolve    bool
)

var schemaIntrospectCmd = &cobra.Command{
	Use:   "introspect",
	Short: "Introspect the live dataset and emit a schema document",
	Long: `Sample documents from the live dataset to infer per-type field shapes.

Caches the result to ~/.config/sanity-cli/schemas/<project>-<dataset>.json
(disable with --no-cache) and prints the schema doc to stdout.

Examples:
  sanity-cli schema introspect --pretty
  sanity-cli schema introspect --sample-size 25 --pretty
  sanity-cli schema introspect --resolve-references --pretty | jq '.data.types | keys'`,
	RunE: runSchemaIntrospect,
}

func init() {
	schemaIntrospectCmd.Flags().IntVar(&introspectSampleSize, "sample-size", 50, "How many docs to sample per type")
	schemaIntrospectCmd.Flags().IntVar(&introspectMaxDepth, "max-depth", 6, "Max recursion depth into nested objects")
	schemaIntrospectCmd.Flags().BoolVar(&introspectNoCache, "no-cache", false, "Skip writing the schema cache to disk")
	schemaIntrospectCmd.Flags().BoolVar(&introspectResolve, "resolve-references", false, "Also resolve reference target _types (slower, extra GROQ call)")
	schemaCmd.AddCommand(schemaIntrospectCmd)
}

func runSchemaIntrospect(cmd *cobra.Command, args []string) error {
	inst, _, _, err := resolveInstance()
	if err != nil {
		return emitError("instance_resolution_failed", err.Error(), nil)
	}
	if err := inst.Validate(); err != nil {
		return emitError("invalid_instance", err.Error(), nil)
	}

	client := api.DefaultClient(inst, timeout)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	doc, err := schema.Introspect(ctx, client, inst, schema.Options{
		SampleSize:        introspectSampleSize,
		MaxDepth:          introspectMaxDepth,
		ResolveReferences: introspectResolve,
	})
	if err != nil {
		return emitError("introspect_failed", err.Error(), nil)
	}

	cachePath := ""
	if !introspectNoCache {
		cp, err := schema.Save(doc)
		if err != nil {
			// Cache failure shouldn't break the introspection result.
			emitWarning("schema_cache_failed", err.Error())
		} else {
			cachePath = cp
		}
	}
	return emitSchemaSuccess(doc, doc.IntrospectedAt, cachePath)
}

// emitWarning prints a structured warning to stderr without affecting exit code.
func emitWarning(code, message string) {
	_, _ = os.Stderr.WriteString("warning: " + code + ": " + message + "\n")
}
