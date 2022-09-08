// Package config contains types and functions for managing the user's local configuration for the command-line interface.
package config

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const (
	configFileName = "config.json"
)

// The Config type describes the structure of the user's local configuration file. These values are used for performing
// operations against the control-plane API.
type Config struct {
	// Organization denotes the user's selected organization.
	Organization string `json:"organization"`
}

// ErrNoConfiguration is the error given when a configuration file cannot be found in the config directory.
var ErrNoConfiguration = errors.New("no configuration file")

// ErrConfigExists is the error given when a configuration file is already present in the config directory.
var ErrConfigExists = errors.New("config exists")

// Load the configuration file from the config directory, decoding it into a Config type. The location of the configuration
// file changes based on the host operating system. See the documentation for os.UserConfigDir for specifics on where
// the config file is loaded from. Returns ErrNoConfiguration if the config file cannot be found.
func Load() (*Config, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	configFile := filepath.Join(configDir, "jsctl", configFileName)
	file, err := os.Open(configFile)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return nil, ErrNoConfiguration
	case err != nil:
		return nil, err
	}
	defer file.Close()

	var config Config
	if err = json.NewDecoder(file).Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// Create a new configuration file in the config directory containing the contents of the given Config type. The location
// of the configuration file changes based on the host operating system. See the documentation for os.UserConfigDir for
// specifics on where the config file is written to. Returns ErrConfigExists if a config file already exists.
func Create(config *Config) error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	jsctlDir := filepath.Join(configDir, "jsctl")
	if _, err = os.Stat(jsctlDir); errors.Is(err, os.ErrNotExist) {
		if err = os.MkdirAll(jsctlDir, 0755); err != nil {
			return err
		}
	}

	configFile := filepath.Join(jsctlDir, configFileName)
	if _, err = os.Stat(configFile); err == nil {
		return ErrConfigExists
	}

	file, err := os.Create(configFile)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(config)
}

// Save the provided configuration, updating an existing file if it already exists. The location of the configuration
// file changes based on the host operating system. See the documentation for os.UserConfigDir for specifics on where
// the config file is written to.
func Save(config *Config) error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	configFile := filepath.Join(configDir, "jsctl", configFileName)
	file, err := os.Create(configFile)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(config)
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
