// Package schema infers a Sanity dataset's shape from sample documents.
package schema

// Doc is the top-level introspected schema document.
type Doc struct {
	ProjectID      string             `json:"project_id"`
	Dataset        string             `json:"dataset"`
	APIVersion     string             `json:"api_version"`
	Perspective    string             `json:"perspective"`
	IntrospectedAt string             `json:"introspected_at"`
	SampleSize     int                `json:"sample_size"`
	MaxDepth       int                `json:"max_depth"`
	Types          map[string]*TypeInfo `json:"types"`
}

// TypeInfo summarizes a single Sanity document type.
type TypeInfo struct {
	Count      int                  `json:"count,omitempty"`
	SampleSize int                  `json:"sample_size"`
	Empty      bool                 `json:"empty,omitempty"`
	Fields     map[string]*Field    `json:"fields,omitempty"`
}

// Field describes the inferred shape of a single field path.
type Field struct {
	Type       string            `json:"type"`             // string|number|boolean|null|datetime|array|object|reference|asset|union
	Optional   bool              `json:"optional,omitempty"`
	Nullable   bool              `json:"nullable,omitempty"`
	Tag        string            `json:"tag,omitempty"`    // portableText|slug|geopoint|...
	AssetKind  string            `json:"asset_kind,omitempty"` // image|file
	To         []string          `json:"to,omitempty"`     // reference targets (best-effort)
	Unresolved bool              `json:"unresolved,omitempty"` // reference target unresolved
	Truncated  bool              `json:"truncated,omitempty"`  // hit max_depth
	Fields     map[string]*Field `json:"fields,omitempty"` // for objects
	Of         []*Field          `json:"of,omitempty"`     // array element shapes (union)
	Blocks     *BlockSummary     `json:"blocks,omitempty"` // for portable text
	Union      []string          `json:"union,omitempty"`  // observed type names when mixed
	// internals
	seenCount  int
	totalCount int
}

// BlockSummary aggregates the unique styles and marks observed in portable text.
type BlockSummary struct {
	Styles []string `json:"styles,omitempty"`
	Marks  []string `json:"marks,omitempty"`
}
