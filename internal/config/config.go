// Package config provides configuration management for the Sanity CLI.
package config

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// DefaultAPIVersion is used when no version is provided.
	DefaultAPIVersion = "2024-10-01"

	// DefaultPerspective is used when no perspective is provided.
	DefaultPerspective = "published"

	// DefaultInstanceName is the name assigned during first-run setup.
	DefaultInstanceName = "default"
)

// Instance represents a single Sanity environment (project + dataset + token).
type Instance struct {
	ProjectID   string `json:"project_id"`
	Dataset     string `json:"dataset"`
	Token       string `json:"token"`
	APIVersion  string `json:"api_version,omitempty"`
	UseCDN      bool   `json:"use_cdn,omitempty"`
	Perspective string `json:"perspective,omitempty"`
}

// Validate checks the instance for required fields.
func (i *Instance) Validate() error {
	if i == nil {
		return errors.New("instance is nil")
	}
	if i.ProjectID == "" {
		return errors.New("project_id is required")
	}
	if i.Dataset == "" {
		return errors.New("dataset is required")
	}
	if i.Token == "" {
		return errors.New("token is required")
	}
	return nil
}

// EffectiveAPIVersion returns the configured api_version or the default.
func (i *Instance) EffectiveAPIVersion() string {
	if i.APIVersion == "" {
		return DefaultAPIVersion
	}
	return i.APIVersion
}

// EffectivePerspective returns the configured perspective or the default.
func (i *Instance) EffectivePerspective() string {
	if i.Perspective == "" {
		return DefaultPerspective
	}
	return i.Perspective
}

// Config holds the application configuration that is persisted to disk.
type Config struct {
	CurrentInstance  string               `json:"current_instance,omitempty"`
	DefaultProjectID string               `json:"default_project_id,omitempty"`
	Instances        map[string]*Instance `json:"instances,omitempty"`
}

// GetConfigDir returns the platform-specific config directory for sanity-cli.
var GetConfigDir = func() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config directory: %w", err)
	}
	return filepath.Join(configDir, "sanity-cli"), nil
}

// GetConfigPath returns the full path to the config file.
func GetConfigPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "config.json"), nil
}

// GetSchemasDir returns the directory used to cache introspected schemas.
func GetSchemasDir() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "schemas"), nil
}

// Load reads the config file. Returns an empty Config if the file doesn't exist.
func Load() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Instances: map[string]*Instance{}}, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	if cfg.Instances == nil {
		cfg.Instances = map[string]*Instance{}
	}

	return &cfg, nil
}

// Save writes the config to disk with secure permissions.
func Save(cfg *Config) error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// PromptForInstance interactively prompts for instance fields.
func PromptForInstance(name string) (*Instance, error) {
	fmt.Fprintln(os.Stderr, "Configuring Sanity instance:", name)
	fmt.Fprintln(os.Stderr, "Tokens are managed at https://www.sanity.io/manage")
	reader := bufio.NewReader(os.Stdin)

	project, err := promptLine(reader, "Project ID")
	if err != nil {
		return nil, err
	}
	dataset, err := promptLine(reader, "Dataset (e.g. production)")
	if err != nil {
		return nil, err
	}
	token, err := promptLine(reader, "API token")
	if err != nil {
		return nil, err
	}
	apiVersion, err := promptLineDefault(reader, "API version", DefaultAPIVersion)
	if err != nil {
		return nil, err
	}

	inst := &Instance{
		ProjectID:   project,
		Dataset:     dataset,
		Token:       token,
		APIVersion:  apiVersion,
		Perspective: DefaultPerspective,
	}
	if err := inst.Validate(); err != nil {
		return nil, err
	}
	return inst, nil
}

func promptLine(r *bufio.Reader, label string) (string, error) {
	fmt.Fprintf(os.Stderr, "%s: ", label)
	v, err := r.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", label, err)
	}
	v = strings.TrimSpace(v)
	if v == "" {
		return "", fmt.Errorf("%s cannot be empty", label)
	}
	return v, nil
}

func promptLineDefault(r *bufio.Reader, label, def string) (string, error) {
	fmt.Fprintf(os.Stderr, "%s [%s]: ", label, def)
	v, err := r.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", label, err)
	}
	v = strings.TrimSpace(v)
	if v == "" {
		return def, nil
	}
	return v, nil
}

// RedactToken returns a redacted form of the token, suitable for display.
func RedactToken(token string) string {
	if token == "" {
		return ""
	}
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "..." + token[len(token)-4:]
}
