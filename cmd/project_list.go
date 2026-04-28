package cmd

import (
	"context"
	"errors"

	"github.com/spf13/cobra"
	"github.com/vstratful/sanity-cli/internal/api"
)

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List projects accessible to the current token",
	RunE:  runProjectList,
}

func init() {
	projectCmd.AddCommand(projectListCmd)
}

func runProjectList(cmd *cobra.Command, args []string) error {
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

	projects, err := client.ListProjects(ctx)
	if err != nil {
		var apiErr *api.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 403 {
			return emitError("manage_api_forbidden",
				"the current token does not have access to the Manage API; use a personal/admin token",
				map[string]interface{}{"status": apiErr.StatusCode})
		}
		return emitError("list_projects_failed", err.Error(), nil)
	}
	return emitSuccess(projects)
}
