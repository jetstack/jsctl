// Package config contains types and functions for managing the user's local configuration for the command-line interface.
package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	configFileName = "config.json"
)

// ContextKey is a type used as a key for the config path in the context
type ContextKey struct{}

// The Config type describes the structure of the user's local configuration file. These values are used for performing
// operations against the control-plane API.
type Config struct {
	// Organization denotes the user's selected organization.
	Organization string `json:"organization"`
}

// ErrNoConfiguration is the error given when a configuration file cannot be found in the config directory.
var ErrNoConfiguration = errors.New("no configuration file")

// Load the configuration file from the config directory specified in the provided context.Context.
// Returns ErrNoConfiguration if the config file cannot be found.
func Load(ctx context.Context) (*Config, error) {
	configDir, ok := ctx.Value(ContextKey{}).(string)
	if !ok {
		return nil, fmt.Errorf("no config path provided")
	}
	configFile := filepath.Join(configDir, configFileName)

	data, err := ReadConfigFile(ctx, configFileName)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return nil, ErrNoConfiguration
	case err != nil:
		return nil, err
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config file %q: %w", configFile, err)
	}

	return &config, nil
}

// Delete will remove the config file if one exists at the path set in the context
func Delete(ctx context.Context) error {
	var err error

	configDir, ok := ctx.Value(ContextKey{}).(string)
	if !ok {
		return fmt.Errorf("no config path provided")
	}
	configFile := filepath.Join(configDir, configFileName)

	_, err = os.Stat(configFile)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	return os.Remove(configFile)
}

// Save the provided configuration, updating an existing file if it already exists.
func Save(ctx context.Context, cfg *Config) error {
	jsonBytes, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %s", err)
	}

	err = WriteConfigFile(ctx, configFileName, jsonBytes)
	if err != nil {
		return fmt.Errorf("failed to write config: %s", err)
	}

	return nil
}

// ReadConfigFile reads a file from the config directory specified in the
// provided context
func ReadConfigFile(ctx context.Context, path string) ([]byte, error) {
	var err error

	configDir, ok := ctx.Value(ContextKey{}).(string)
	if !ok {
		return []byte{}, fmt.Errorf("no config path provided")
	}
	configFile := filepath.Join(configDir, path)

	file, err := os.Open(configFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to read config file %q: %w", configFile, err)
	}

	return data, nil
}

// WriteConfigFile writes a file with the correct permissions to the
// config directory specified in the provided context
func WriteConfigFile(ctx context.Context, path string, data []byte) error {
	var err error

	configDir, ok := ctx.Value(ContextKey{}).(string)
	if !ok {
		return fmt.Errorf("no config path provided")
	}
	configFile := filepath.Join(configDir, path)

	err = os.WriteFile(configFile, data, 0600)
	if err != nil {
		return fmt.Errorf("failed to write config %q: %w", configFile, err)
	}

	return nil
}

type ctxKey struct{}

// ToContext returns a context.Context that contains the provided Config instance.
func ToContext(ctx context.Context, config *Config) context.Context {
	return context.WithValue(ctx, ctxKey{}, config)
}

// FromContext attempts to obtain a Config type contained within the provided context.Context. The boolean return value
// is used to indicate if the context contains a Config.
func FromContext(ctx context.Context) (*Config, bool) {
	value := ctx.Value(ctxKey{})
	if value == nil {
		return nil, false
	}

	config, ok := value.(*Config)
	return config, ok
}
