package registry_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jetstack/jsctl/internal/client"
	"github.com/jetstack/jsctl/internal/config"
	"github.com/jetstack/jsctl/internal/registry"
	"github.com/jetstack/jsctl/internal/subscription"
)

func TestRegistryAuthInit(t *testing.T) {
	httpClient := &MockHTTPClient{
		Response: []subscription.GoogleServiceAccountWithKey{
			{
				GoogleServiceAccount: subscription.GoogleServiceAccount{DisplayName: "things"},
				Key:                  subscription.GoogleServiceAccountKey{PrivateData: "MQo="},
			},
		},
	}

	// tempConfigDir is created in order to test that credentials are put in the correct place
	tempConfigDir, err := os.MkdirTemp("", "registry-test-*")
	require.NoError(t, err)
	defer os.Remove(tempConfigDir)

	ctx := config.ToContext(context.Background(), &config.Config{Organization: "example"})
	ctx = context.WithValue(ctx, config.ContextKey{}, tempConfigDir)

	bytes, err := registry.FetchOrLoadJetstackSecureEnterpriseRegistryCredentials(ctx, httpClient)
	require.NoError(t, err)
	assert.Equal(t, "1\n", string(bytes))

	// call it again to make sure that the file is reused
	bytes, err = registry.FetchOrLoadJetstackSecureEnterpriseRegistryCredentials(ctx, httpClient)
	require.NoError(t, err)
	assert.Equal(t, "1\n", string(bytes))

	// check that the contents on disk is also correct
	bytes, err = os.ReadFile(tempConfigDir + "/eu.gcr.io--jetstack-secure-enterprise.json")
	require.NoError(t, err)
	assert.Equal(t, "1\n", string(bytes))

	// the http handler for the call should have only been invoked once
	assert.Equal(t, 1, httpClient.InvocationCount)
}

// MockHTTPClient is a mock HTTP client used to return prepared responses for
// API calls
type MockHTTPClient struct {
	Method          string
	URI             string
	Body            interface{}
	Response        interface{}
	InvocationCount int
}

func (m *MockHTTPClient) Do(_ context.Context, method, uri string, body, out interface{}) error {
	m.InvocationCount++

	m.URI = uri
	m.Method = method
	m.Body = body

	if m.Response == nil {
		return nil
	}

	if err, ok := m.Response.(client.APIError); ok {
		return err
	}

	if out == nil {
		return nil
	}

	data, err := json.Marshal(m.Response)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, out)
}
