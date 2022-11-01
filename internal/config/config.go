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

// DefaultConfigDir returns the preferred config directory for the current platform
func DefaultConfigDir() (string, error) {
	dir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %s", err)
	}

	dir = filepath.Join(dir, ".jsctl")

	return dir, nil
}

// MigrateDefaultConfig has been implemented in response to:
// https://github.com/jetstack/jsctl/issues/52
// We were using UserConfigDir from the os package, however, users did not
// expect this. So we have moved to ~/.jsctl or something equivalent instead.
func MigrateDefaultConfig(newConfigDir string) error {
	legacyDirs, err := legacyConfigDirs()
	if err != nil {
		return fmt.Errorf("legacy config dirs needed in migration: %s", err)
	}

	// there might be many legacy dirs to try, however, we can only migrate at
	// most one. If newConfigDir is present, then we will not overwrite it.
	for _, legacyDir := range legacyDirs {
		if _, err := os.Stat(legacyDir); os.IsNotExist(err) {
			// then there is no work to do
			continue
		}

		if _, err := os.Stat(newConfigDir); !os.IsNotExist(err) {
			// then we can't continue because we don't want to overwrite the new config dir
			return fmt.Errorf("config dir %q already exists, please remove either %q or %q", newConfigDir, newConfigDir, legacyDir)
		}

		// move the config to the new dir
		err := os.Rename(legacyDir, newConfigDir)
		if err != nil {
			return fmt.Errorf("failed to move config dir from %q to %q: %s", legacyDir, newConfigDir, err)
		}

		fmt.Fprintf(os.Stderr, "Migrated config from %q to %q\n", legacyDir, newConfigDir)
	}

	return nil
}

// legacyConfigDir returns the possible legacy config directory for the
// current platform which might have been used in a previous version of jsctl.
// Currently, this only returns the value of UserConfigDir()
func legacyConfigDirs() ([]string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to determine the legacy config directory: %s", err)
	}

	configDir = filepath.Join(configDir, "jsctl")

	return []string{configDir}, nil
}

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
		return nil, fmt.Errorf("no config path provided")
	}
	configFile := filepath.Join(configDir, path)

	// check that the file is not a symlink
	// https://github.com/jetstack/jsctl/issues/43
	configFileInfo, err := os.Lstat(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to stat config file %q: %w", configFile, err)
	}
	if configFileInfo.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("config file %q is a symlink, refusing to read", configFile)
	}

	// check the file permissions and update them if not 0600
	if configFileInfo.Mode().Perm() != 0600 {
		// TODO: we should error here in future. This is here to gracefully
		// handle config files from older versions
		fmt.Fprintf(os.Stderr, "warning: config file %q has insecure file permissions, correcting them\n", configFile)
		err = os.Chmod(configFile, 0600)
		if err != nil {
			return nil, fmt.Errorf("failed to correct config file permissions for %q", configFile)
		}
	}

	file, err := os.Open(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file %q: %w", configFile, err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %q: %w", configFile, err)
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

	// check that the file is not a symlink
	// https://github.com/jetstack/jsctl/issues/43
	configFileInfo, err := os.Lstat(configFile)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to stat config file %q: %w", configFile, err)
	}
	if err == nil { // file exists
		if configFileInfo.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("config file %q is a symlink, refusing to write", configFile)
		}
		// check the file permissions and update them if not 0600
		if configFileInfo.Mode().Perm() != 0600 {
			// TODO: we should error here in future. This is here to gracefully
			// handle config files from older versions
			fmt.Fprintf(os.Stderr, "warning: config file %q has insecure file permissions, correcting them\n", configFile)
			err = os.Chmod(configFile, 0600)
			if err != nil {
				return fmt.Errorf("failed to correct config file permissions for %q", configFile)
			}
		}
	}

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
