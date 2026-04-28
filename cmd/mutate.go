package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vstratful/sanity-cli/internal/api"
)

var (
	mutateConfirm        bool
	mutateDryRun         bool
	mutateTransactionID  string
	mutateReturnIDs      bool
	mutateReturnDocs     bool
	mutateAutoGenKeys    bool
)

// validMutationKeys lists the operations Sanity recognises in a mutation entry.
var validMutationKeys = []string{
	"create", "createOrReplace", "createIfNotExists", "patch", "delete",
}

var mutateCmd = &cobra.Command{
	Use:   "mutate [file]",
	Short: "Apply mutations from a JSON file or stdin (requires --confirm)",
	Long: `Apply a list of Sanity mutations to the active dataset.

Input is a JSON array where each element is a single mutation object with
exactly one of: create, createOrReplace, createIfNotExists, patch, delete.

Pass "-" or omit the file to read from stdin.

Mutations are refused unless --confirm is set OR SANITY_CLI_AUTO_CONFIRM=1 in
the environment. --dry-run always supersedes; the mutation is parsed and
previewed but never sent.

Examples:
  sanity-cli mutate ./mutations.json --confirm
  echo '[{"create":{"_type":"note","title":"hi"}}]' | sanity-cli mutate - --dry-run
  SANITY_CLI_AUTO_CONFIRM=1 sanity-cli mutate ./mutations.json`,
	Args: cobra.MaximumNArgs(1),
	RunE: runMutate,
}

func init() {
	mutateCmd.Flags().BoolVar(&mutateConfirm, "confirm", false, "Required to actually apply mutations")
	mutateCmd.Flags().BoolVar(&mutateDryRun, "dry-run", false, "Parse and preview without sending")
	mutateCmd.Flags().StringVar(&mutateTransactionID, "transaction-id", "", "Optional transaction ID")
	mutateCmd.Flags().BoolVar(&mutateReturnIDs, "return-ids", false, "Ask the API to return mutated document IDs")
	mutateCmd.Flags().BoolVar(&mutateReturnDocs, "return-documents", false, "Ask the API to return full mutated documents")
	mutateCmd.Flags().BoolVar(&mutateAutoGenKeys, "auto-generate-array-keys", false, "Tell the API to auto-generate array element keys")
	rootCmd.AddCommand(mutateCmd)
}

func runMutate(cmd *cobra.Command, args []string) error {
	source := "-"
	if len(args) == 1 {
		source = args[0]
	}

	mutations, err := readMutations(source)
	if err != nil {
		return emitError("invalid_input", err.Error(), nil)
	}

	counts, firstObj := summarizeMutations(mutations)

	autoConfirm := envTrue("SANITY_CLI_AUTO_CONFIRM")

	if mutateDryRun {
		return emitSuccess(map[string]interface{}{
			"applied": false,
			"dry_run": true,
			"preview": map[string]interface{}{
				"count":  len(mutations),
				"counts": counts,
				"first":  firstObj,
			},
		})
	}

	if !mutateConfirm && !autoConfirm {
		msg := fmt.Sprintf("Refusing to apply %d mutations without --confirm or SANITY_CLI_AUTO_CONFIRM=1", len(mutations))
		return emitError("confirmation_required", msg, map[string]interface{}{
			"count":  len(mutations),
			"counts": counts,
			"first":  firstObj,
		})
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

	resp, err := client.Mutate(ctx, mutations, &api.MutateOptions{
		TransactionID:    mutateTransactionID,
		ReturnIDs:        mutateReturnIDs,
		ReturnDocuments:  mutateReturnDocs,
		AutoGenerateKeys: mutateAutoGenKeys,
	})
	if err != nil {
		return emitError("mutate_failed", err.Error(), nil)
	}

	out := map[string]interface{}{
		"applied":         true,
		"transaction_id":  resp.TransactionID,
		"count":           len(mutations),
	}
	if len(resp.Results) > 0 {
		var v any
		if jerr := json.Unmarshal(resp.Results, &v); jerr == nil {
			out["results"] = v
		}
	}
	if len(resp.Documents) > 0 {
		var v any
		if jerr := json.Unmarshal(resp.Documents, &v); jerr == nil {
			out["documents"] = v
		}
	}
	return emitSuccess(out)
}

// readMutations parses the input file (or stdin when source == "-") into a
// slice of raw mutation objects, validating that each element is an object
// containing exactly one recognised mutation key.
func readMutations(source string) ([]json.RawMessage, error) {
	var data []byte
	var err error
	if source == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(source)
	}
	if err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, errors.New("input is empty")
	}

	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err != nil {
		return nil, fmt.Errorf("input must be a JSON array of mutation objects: %w", err)
	}
	if len(arr) == 0 {
		return nil, errors.New("input array is empty")
	}

	for i, raw := range arr {
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(raw, &obj); err != nil {
			return nil, fmt.Errorf("mutation %d is not a JSON object: %w", i, err)
		}
		matches := 0
		for _, key := range validMutationKeys {
			if _, ok := obj[key]; ok {
				matches++
			}
		}
		if matches != 1 {
			return nil, fmt.Errorf("mutation %d must contain exactly one of: %s", i, strings.Join(validMutationKeys, ", "))
		}
	}
	return arr, nil
}

func summarizeMutations(mutations []json.RawMessage) (map[string]int, json.RawMessage) {
	counts := map[string]int{}
	for _, raw := range mutations {
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(raw, &obj); err != nil {
			continue
		}
		for _, key := range validMutationKeys {
			if _, ok := obj[key]; ok {
				counts[key]++
				break
			}
		}
	}
	var first json.RawMessage
	if len(mutations) > 0 {
		first = mutations[0]
	}
	return counts, first
}

func envTrue(name string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
	return v != "" && v != "0" && v != "false" && v != "no"
}
