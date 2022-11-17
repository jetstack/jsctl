package backup

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

func TestBackup(t *testing.T) {
	expectedBackupYAML, err := os.ReadFile("fixtures/backup.yaml")
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		w.Header().Set("Content-Type", "application/json")

		var data []byte
		switch r.URL.Path {
		// CRDs are needed to determine the issuer types to backup
		case "/apis/apiextensions.k8s.io/v1/customresourcedefinitions":
			data, err = os.ReadFile("fixtures/crd-list.json")
			require.NoError(t, err)
		case "/apis/cert-manager.io/v1/certificates":
			data, err = os.ReadFile("fixtures/certificate-list.json")
			require.NoError(t, err)
		case "/apis/cert-manager.io/v1/clusterissuers":
			data, err = os.ReadFile("fixtures/cluster-issuer-list.json")
			require.NoError(t, err)
		case "/apis/cert-manager.io/v1/issuers":
			data, err = os.ReadFile("fixtures/issuer-list.json")
			require.NoError(t, err)
		case "/apis/cas-issuer.jetstack.io/v1beta1/googlecasissuers":
			data, err = os.ReadFile("fixtures/googlecasissuer-list.json")
			require.NoError(t, err)
		case "/apis/cas-issuer.jetstack.io/v1beta1/googlecasclusterissuers":
			data = []byte(`{"items": []}`)
			require.NoError(t, err)
		case "/apis/awspca.cert-manager.io/v1beta1/awspcaissuers":
			data, err = os.ReadFile("fixtures/awspcaissuer-list.json")
			require.NoError(t, err)
		case "/apis/policy.cert-manager.io/v1alpha1/certificaterequestpolicies":
			data, err = os.ReadFile("fixtures/certificate-request-policy-list.json")
			require.NoError(t, err)
		default:
			t.Fatalf("unexpected request: %s", r.URL.Path)
		}

		w.Write(data)
	}))

	opts := ClusterBackupOptions{
		RestConfig: &rest.Config{Host: server.URL},

		FormatResources: true,

		IncludeCertificates:               true,
		IncludeCertificateRequestPolicies: true,
		IncludeIssuers:                    true,
	}

	backup, err := FetchClusterBackup(context.Background(), opts)
	require.NoError(t, err)

	backupYAML, err := backup.ToYAML()
	require.NoError(t, err)

	assert.Equal(t, string(expectedBackupYAML), string(backupYAML))
}
