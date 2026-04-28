package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vstratful/sanity-cli/internal/api"
)

var (
	assetType  string
	assetLabel string
	assetTitle string
)

var assetCmd = &cobra.Command{
	Use:   "asset",
	Short: "Manage Sanity assets",
}

var assetUploadCmd = &cobra.Command{
	Use:   "upload <path>",
	Short: "Upload a binary as an image or file asset",
	Long: `Upload a file at <path> to the active dataset. Use --type image or --type file
to control which assets endpoint is used. Content type is inferred from the
file extension; override via --content-type if needed.`,
	Args: cobra.ExactArgs(1),
	RunE: runAssetUpload,
}

var assetContentType string

func init() {
	assetUploadCmd.Flags().StringVar(&assetType, "type", "image", "Asset kind: image|file")
	assetUploadCmd.Flags().StringVar(&assetLabel, "label", "", "Optional label")
	assetUploadCmd.Flags().StringVar(&assetTitle, "title", "", "Optional title")
	assetUploadCmd.Flags().StringVar(&assetContentType, "content-type", "", "Override the inferred Content-Type")
	assetCmd.AddCommand(assetUploadCmd)
	rootCmd.AddCommand(assetCmd)
}

func runAssetUpload(cmd *cobra.Command, args []string) error {
	path := args[0]
	kind := api.AssetKind(strings.ToLower(assetType))
	if kind != api.AssetKindImage && kind != api.AssetKindFile {
		return emitError("invalid_asset_type", "--type must be 'image' or 'file'", nil)
	}

	f, err := os.Open(path)
	if err != nil {
		return emitError("file_open_failed", err.Error(), nil)
	}
	defer f.Close()

	contentType := assetContentType
	if contentType == "" {
		ext := filepath.Ext(path)
		if ct := mime.TypeByExtension(ext); ct != "" {
			contentType = ct
		} else {
			contentType = "application/octet-stream"
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

	doc, err := client.UploadAsset(ctx, kind, f, &api.AssetUploadOptions{
		ContentType: contentType,
		Filename:    filepath.Base(path),
		Label:       assetLabel,
		Title:       assetTitle,
	})
	if err != nil {
		return emitError("asset_upload_failed", err.Error(), nil)
	}

	var docMap any
	if jerr := json.Unmarshal(doc, &docMap); jerr != nil {
		return emitError("decode_failed", fmt.Sprintf("decoding asset doc: %v", jerr), nil)
	}
	return emitSuccess(map[string]interface{}{
		"kind":     string(kind),
		"document": docMap,
	})
}
