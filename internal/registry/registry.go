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
	_, err := config.ReadConfigFile(ctx, fmt.Sprintf("%s.json", jetstackSecureRegistryFileKey))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("error reading registry credentials: %s", err)
	} else if errors.Is(err, os.ErrNotExist) {
		return "not authenticated", nil
	}
	return "authenticated", nil
}

// PathJetstackSecureEnterpriseRegistry will return the path where the credentials for the registry are located
func PathJetstackSecureEnterpriseRegistry(ctx context.Context) (string, error) {
	configDir, ok := ctx.Value(config.ContextKey{}).(string)
	if !ok {
		return "", fmt.Errorf("no config directory found in context")
	}

	return filepath.Join(configDir, fmt.Sprintf("%s.json", jetstackSecureRegistryFileKey)), nil
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

	data, err := config.ReadConfigFile(ctx, fmt.Sprintf("%s.json", jetstackSecureRegistryFileKey))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("error reading registry credentials: %s", err)
	}
	if err == nil {
		return data, nil
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

	err = config.WriteConfigFile(ctx, fmt.Sprintf("%s.json", jetstackSecureRegistryFileKey), registryCredentialsBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to write registry credentials to path %q: %w", registryCredentialsPath, err)
	}

	return registryCredentialsBytes, nil
}
