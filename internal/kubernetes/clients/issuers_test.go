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

func TestListSupportedIssuers(t *testing.T) {
	expectedSupportedIssuers := SupportedIssuerList{
		{
			CRDName:  "issuers.cert-manager.io",
			Versions: []string{"v1"},
		},
		{
			CRDName:  "clusterissuers.cert-manager.io",
			Versions: []string{"v1"},
		},
		{
			CRDName:  "venafiissuers.jetstack.io",
			Versions: []string{"v1alpha1"},
		},
		{
			CRDName:  "venaficlusterissuers.jetstack.io",
			Versions: []string{"v1alpha1"},
		},
		{
			CRDName:  "awspcaissuers.awspca.cert-manager.io",
			Versions: []string{"v1beta1"},
		},
		{
			CRDName:  "awspcaclusterissuers.awspca.cert-manager.io",
			Versions: []string{"v1beta1"},
		},
		{
			CRDName:  "kmsissuers.cert-manager.skyscanner.net",
			Versions: []string{"v1alpha1"},
		},
		{
			CRDName:  "googlecasissuers.cas-issuer.jetstack.io",
			Versions: []string{"v1beta1"},
		},
		{
			CRDName:  "googlecasclusterissuers.cas-issuer.jetstack.io",
			Versions: []string{"v1beta1"},
		},
		{
			CRDName:  "originissuers.cert-manager.k8s.cloudflare.com",
			Versions: []string{"v1"},
		},
		{
			CRDName:  "stepissuers.certmanager.step.sm",
			Versions: []string{"v1beta1"},
		},
		{
			CRDName:  "stepclusterissuers.certmanager.step.sm",
			Versions: []string{"v1beta1"},
		},
	}

	result, err := ListSupportedIssuers()
	require.NoError(t, err)
	assert.Equal(t, expectedSupportedIssuers, result)
}

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
