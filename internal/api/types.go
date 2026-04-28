package api

import "encoding/json"

// QueryResponse is the envelope returned by the GROQ query endpoint.
type QueryResponse struct {
	Result json.RawMessage `json:"result"`
	Query  string          `json:"query,omitempty"`
	Ms     int             `json:"ms,omitempty"`
}

// AssetKind distinguishes image assets from generic file assets.
type AssetKind string

const (
	AssetKindImage AssetKind = "image"
	AssetKindFile  AssetKind = "file"
)

// AssetUploadOptions configures an asset upload request.
type AssetUploadOptions struct {
	ContentType string
	Filename    string
	Label       string
	Title       string
}

// AssetResponse is the envelope returned by the asset upload endpoint.
type AssetResponse struct {
	Document json.RawMessage `json:"document"`
}

// MutateOptions configures a mutate request.
type MutateOptions struct {
	TransactionID    string
	ReturnIDs        bool
	ReturnDocuments  bool
	Visibility       string // "sync" | "async" | "deferred"
	DryRun           bool
	AutoGenerateKeys bool
}

// MutateResponse is the envelope returned by the mutate endpoint.
type MutateResponse struct {
	TransactionID string          `json:"transactionId,omitempty"`
	Results       json.RawMessage `json:"results,omitempty"`
	Documents     json.RawMessage `json:"documents,omitempty"`
}

// Project represents a Sanity project as returned by the Manage API.
type Project struct {
	ID           string          `json:"id"`
	DisplayName  string          `json:"displayName"`
	StudioHost   string          `json:"studioHost,omitempty"`
	Organization string          `json:"organizationId,omitempty"`
	Metadata     json.RawMessage `json:"metadata,omitempty"`
	Members      json.RawMessage `json:"members,omitempty"`
}

// Dataset represents a dataset within a project.
type Dataset struct {
	Name       string `json:"name"`
	ACLMode    string `json:"aclMode,omitempty"`
	AddonFor   string `json:"addonFor,omitempty"`
	Tags       []any  `json:"tags,omitempty"`
	CreatedAt  string `json:"createdAt,omitempty"`
	DatasetID  string `json:"datasetId,omitempty"`
	ProjectID  string `json:"projectId,omitempty"`
}
