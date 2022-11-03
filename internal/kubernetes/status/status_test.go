package status

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"
)

func TestGatherClusterPreInstallStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// read the contents of fixtures files

		data, err := os.ReadFile("fixtures/crd-list.json")
		require.NoError(t, err)

		w.Write(data)
	}))

	cfg := &rest.Config{
		Host: server.URL,
	}

	status, err := GatherClusterPreInstallStatus(context.Background(), cfg)
	require.NoError(t, err)

	assert.Equal(t, status, &ClusterPreInstallStatus{
		CRDGroups: []crdGroup{
			{
				Name: "cert-manager.io",
				CRDs: []string{
					"certificaterequestpolicies.policy.cert-manager.io",
					"certificaterequests.cert-manager.io",
					"certificates.cert-manager.io",
					"challenges.acme.cert-manager.io",
					"clusterissuers.cert-manager.io",
					"issuers.cert-manager.io",
					"orders.acme.cert-manager.io",
				},
			},
			{
				Name: "jetstack.io",
				CRDs: []string{
					"installations.operator.jetstack.io",
				},
			},
		},
	})
}
