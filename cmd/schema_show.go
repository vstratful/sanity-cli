package cmd

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/vstratful/sanity-cli/internal/api"
	"github.com/vstratful/sanity-cli/internal/schema"
)

var schemaShowRefresh bool

var schemaShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Print the cached schema for the current instance",
	RunE:  runSchemaShow,
}

func init() {
	schemaShowCmd.Flags().BoolVar(&schemaShowRefresh, "refresh", false, "Re-introspect first if the cache is stale or missing")
	schemaCmd.AddCommand(schemaShowCmd)
}

func runSchemaShow(cmd *cobra.Command, args []string) error {
	inst, _, _, err := resolveInstance()
	if err != nil {
		return emitError("instance_resolution_failed", err.Error(), nil)
	}
	if err := inst.Validate(); err != nil {
		return emitError("invalid_instance", err.Error(), nil)
	}

	doc, path, err := schema.Load(inst.ProjectID, inst.Dataset)
	if err != nil || schemaShowRefresh {
		if !schemaShowRefresh && err != nil {
			// Cache miss: refresh implicitly is heavier; require --refresh explicitly.
			return emitError("schema_cache_missing",
				"no cached schema; run `sanity-cli schema introspect` (or pass --refresh)",
				map[string]string{"cache_path": path})
		}
		client := api.DefaultClient(inst, timeout)
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		doc, err = schema.Introspect(ctx, client, inst, schema.Options{})
		if err != nil {
			return emitError("introspect_failed", err.Error(), nil)
		}
		path, err = schema.Save(doc)
		if err != nil {
			emitWarning("schema_cache_failed", err.Error())
		}
	}
	return emitSchemaSuccess(doc, doc.IntrospectedAt, path)
}
