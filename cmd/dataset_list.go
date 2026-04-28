package cmd

import (
	"context"
	"errors"

	"github.com/spf13/cobra"
	"github.com/vstratful/sanity-cli/internal/api"
	"github.com/vstratful/sanity-cli/internal/config"
)

var datasetListProject string

var datasetListCmd = &cobra.Command{
	Use:   "list",
	Short: "List datasets in a project (defaults to the current instance's project)",
	RunE:  runDatasetList,
}

func init() {
	datasetListCmd.Flags().StringVar(&datasetListProject, "project", "", "Project ID to list datasets for (defaults to current instance / default_project_id)")
	datasetCmd.AddCommand(datasetListCmd)
}

func runDatasetList(cmd *cobra.Command, args []string) error {
	inst, cfg, _, err := resolveInstance()
	if err != nil {
		return emitError("instance_resolution_failed", err.Error(), nil)
	}
	if err := inst.Validate(); err != nil {
		return emitError("invalid_instance", err.Error(), nil)
	}

	projectID := datasetListProject
	if projectID == "" {
		projectID = inst.ProjectID
	}
	if projectID == "" && cfg != nil {
		projectID = cfg.DefaultProjectID
	}
	if projectID == "" {
		return emitError("missing_project_id", "no project ID; pass --project or configure an instance", nil)
	}
	_ = config.DefaultAPIVersion // keep import

	client := api.DefaultClient(inst, timeout)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	datasets, err := client.ListDatasets(ctx, projectID)
	if err != nil {
		var apiErr *api.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 403 {
			return emitError("manage_api_forbidden",
				"the current token does not have access to the Manage API for this project",
				map[string]interface{}{"status": apiErr.StatusCode, "project_id": projectID})
		}
		return emitError("list_datasets_failed", err.Error(), nil)
	}
	return emitSuccess(map[string]interface{}{
		"project_id": projectID,
		"datasets":   datasets,
	})
}
