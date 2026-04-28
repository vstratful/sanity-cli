package schema

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vstratful/sanity-cli/internal/config"
)

// CachePath returns the on-disk cache path for a given project+dataset pair.
func CachePath(projectID, dataset string) (string, error) {
	dir, err := config.GetSchemasDir()
	if err != nil {
		return "", err
	}
	name := fmt.Sprintf("%s-%s.json", projectID, dataset)
	return filepath.Join(dir, name), nil
}

// Save writes the schema doc to its on-disk cache, creating directories.
func Save(doc *Doc) (string, error) {
	dir, err := config.GetSchemasDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("creating schemas dir: %w", err)
	}
	path, err := CachePath(doc.ProjectID, doc.Dataset)
	if err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling schema: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return "", fmt.Errorf("writing schema cache: %w", err)
	}
	return path, nil
}

// Load reads a previously cached schema doc.
func Load(projectID, dataset string) (*Doc, string, error) {
	path, err := CachePath(projectID, dataset)
	if err != nil {
		return nil, "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, path, err
	}
	var doc Doc
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, path, fmt.Errorf("parsing cached schema: %w", err)
	}
	return &doc, path, nil
}
