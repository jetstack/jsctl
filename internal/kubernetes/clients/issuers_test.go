package clients

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"
)

func TestAllIssuers_ListKinds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		file, err := os.Open("fixtures/crd-list.json")
		require.NoError(t, err)
		_, err = io.Copy(w, file)
		require.NoError(t, err)
	}))

	cfg := &rest.Config{
		Host: server.URL,
	}
	client, err := NewAllIssuers(cfg)
	require.NoError(t, err)

	foundIssuers, err := client.ListKinds(context.Background())
	require.NoError(t, err)

	assert.ElementsMatch(t,
		[]AnyIssuer{
			CertManagerIssuer,
			CertManagerClusterIssuer,
			GoogleCASIssuer,
			GoogleCASClusterIssuer,
			AWSPCAIssuer,
		},
		foundIssuers,
	)
}
