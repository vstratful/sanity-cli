package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vstratful/sanity-cli/internal/api"
)

var (
	queryParamsFlag string
	queryRaw        bool
)

var queryCmd = &cobra.Command{
	Use:   "query <groq>",
	Short: "Run a GROQ query against the active dataset",
	Long: `Run a GROQ query against the active instance's dataset and print the result as JSON.

Examples:
  sanity-cli query '*[_type == "post"][0..5]{_id,title}' --pretty
  sanity-cli query '*[_type == $t][0]' --params '{"t":"post"}'
  sanity-cli query '*[_type == "post"][0..2]' --raw | jq '.'`,
	Args: cobra.ExactArgs(1),
	RunE: runQuery,
}

func init() {
	queryCmd.Flags().StringVar(&queryParamsFlag, "params", "", "GROQ params as a JSON object (e.g. '{\"t\":\"post\"}')")
	queryCmd.Flags().BoolVar(&queryRaw, "raw", false, "Print raw GROQ result without the success envelope")
	rootCmd.AddCommand(queryCmd)
}

func runQuery(cmd *cobra.Command, args []string) error {
	groq := args[0]

	var params map[string]any
	if queryParamsFlag != "" {
		if err := json.Unmarshal([]byte(queryParamsFlag), &params); err != nil {
			return emitError("invalid_params", fmt.Sprintf("--params must be a JSON object: %v", err), nil)
		}
	}

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

	result, err := client.Query(ctx, groq, params)
	if err != nil {
		return emitError("query_failed", err.Error(), nil)
	}

	if queryRaw {
		var anyResult any
		if err := json.Unmarshal(result, &anyResult); err != nil {
			return emitError("decode_failed", err.Error(), nil)
		}
		return emitSuccessRaw(anyResult)
	}

	var anyResult any
	if err := json.Unmarshal(result, &anyResult); err != nil {
		return emitError("decode_failed", err.Error(), nil)
	}
	return emitSuccess(anyResult)
}
