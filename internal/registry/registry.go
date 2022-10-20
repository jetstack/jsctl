package registry

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/jetstack/jsctl/internal/client"
	"github.com/jetstack/jsctl/internal/config"
	"github.com/jetstack/jsctl/internal/subscription"
)

// jetstackSecureRegistryFileKey is the filename used to store the registry
// credentials in jsctl config
const jetstackSecureRegistryFileKey = "eu.gcr.io--jetstack-secure-enterprise"

// StatusJetstackSecureEnterpriseRegistry will return the status of the registry
// credentials for the Jetstack Secure Enterprise registry stashed to disk
func StatusJetstackSecureEnterpriseRegistry(ctx context.Context) (string, error) {
	configDir, ok := ctx.Value(config.ContextKey{}).(string)
	if !ok {
		return "", fmt.Errorf("no config directory found in context")
	}

	registryCredentialsPath := filepath.Join(configDir, fmt.Sprintf("%s.json", jetstackSecureRegistryFileKey))

	_, err := os.Stat(registryCredentialsPath)
	if errors.Is(err, os.ErrNotExist) {
		return "not authenticated", nil
	}
	if err != nil {
		return "", fmt.Errorf("error checking if registry credentials exist: %s", err)
	}

	return "authenticated", nil
}

// FetchOrLoadJetstackSecureEnterpriseRegistryCredentials will check of there are
// a local copy of registry credentials. If there is, then these are returned,
// if not, then a new set is fetched and stashed in the jsctl config dir specified
func FetchOrLoadJetstackSecureEnterpriseRegistryCredentials(ctx context.Context, httpClient subscription.HTTPClient) ([]byte, error) {
	var err error

	configDir, ok := ctx.Value(config.ContextKey{}).(string)
	if !ok {
		return nil, fmt.Errorf("no config directory found in context")
	}

	registryCredentialsPath := filepath.Join(configDir, fmt.Sprintf("%s.json", jetstackSecureRegistryFileKey))

	_, err = os.Stat(registryCredentialsPath)
	if !errors.Is(err, os.ErrNotExist) {
		// then we can just load and return the file
		bytes, err := os.ReadFile(registryCredentialsPath)
		if err != nil {
			return nil, fmt.Errorf("error reading registry credentials file: %s", err)
		}

		return bytes, nil
	}

	cnf, ok := config.FromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("error getting config from context")
	}

	// organization must be set here so that we know which org to create
	// the credentials in
	if cnf.Organization == "" {
		return nil, fmt.Errorf("no organization must be set")
	}

	serviceAccounts, err := subscription.CreateGoogleServiceAccount(
		ctx,
		httpClient,
		cnf.Organization,
		fmt.Sprintf("%s-jsctl-auto", cnf.Organization),
	)
	if apiErr, ok := err.(client.APIError); ok {
		if apiErr.Status == http.StatusUnauthorized {
			return nil, fmt.Errorf("failed to create registry credentials, current organization %q does not have permissions to access the Jetstack Secure Enterprise registry. Please contact support if this is unexpected.", cnf.Organization)
		}
	}
	if err != nil || len(serviceAccounts) < 1 {
		return nil, fmt.Errorf("failed to create registry credentials: %w", err)
	}

	registryCredentialsBytes, err := base64.StdEncoding.DecodeString(serviceAccounts[0].Key.PrivateData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode registry credentials: %w", err)
	}

	// stash the bytes in the config dir for use in future invocations
	err = os.WriteFile(registryCredentialsPath, registryCredentialsBytes, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to write registry credentials to path %q: %w", registryCredentialsPath, err)
	}

	return registryCredentialsBytes, nil
}
