// Package subscription contains functions to call endpoints in the JSCP
// subscription API
package subscription

import (
	"context"
	_ "embed"
	"net/http"
	"path"
)

type (
	// The HTTPClient interface describes types that perform HTTP requests.
	HTTPClient interface {
		Do(ctx context.Context, method, uri string, body, out interface{}) error
	}

	// GoogleServiceAccount is the base type for Google Service Account data
	// returned by the subscription API
	GoogleServiceAccount struct {
		ID          string `json:"id"`
		DisplayName string `json:"displayName"`
	}

	// GoogleServiceAccountWithKey adds the key to GoogleServiceAccount
	GoogleServiceAccountWithKey struct {
		GoogleServiceAccount
		Key GoogleServiceAccountKey `json:"key"`
	}

	// GoogleServiceAccountKey represents the type of for the GCP SA key and
	// docker config as returned by the subscription API
	GoogleServiceAccountKey struct {
		// PrivateData is the service account credentials encoded in base64
		PrivateData string `json:"privateData"`
		// DockerConfig is a base64 encoded dockerconfig file using the service account credentials to access Jetstack's enterprise registries.
		DockerConfig string `json:"dockerConfig"`
	}
)

// CreateGoogleServiceAccount calls the subscription API to create a new Google
// Service Account for GCR access
func CreateGoogleServiceAccount(ctx context.Context, httpClient HTTPClient, organization, name string) (*[]GoogleServiceAccountWithKey, error) {
	request := []GoogleServiceAccount{
		{DisplayName: name},
	}

	uri := path.Join("/subscription/api/v1/org/", organization, "svc_accounts")

	var serviceAccounts []GoogleServiceAccountWithKey
	if err := httpClient.Do(ctx, http.MethodPost, uri, request, &serviceAccounts); err != nil {
		return nil, err
	}

	return &serviceAccounts, nil
}
