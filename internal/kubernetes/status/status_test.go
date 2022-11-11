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

	"github.com/jetstack/jsctl/internal/kubernetes/status/components"
)

func TestGatherClusterPreInstallStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		w.Header().Set("Content-Type", "application/json")
		// read the contents of fixtures files

		var data []byte
		switch r.URL.Path {
		case "/api/v1/namespaces":
			data, err = os.ReadFile("fixtures/namespace-list.json")
			require.NoError(t, err)
		case "/api/v1/pods":
			data, err = os.ReadFile("fixtures/pod-list.json")
			require.NoError(t, err)
		case "/apis/apiextensions.k8s.io/v1/customresourcedefinitions":
			data, err = os.ReadFile("fixtures/crd-list.json")
			require.NoError(t, err)
		case "/apis/networking.k8s.io/v1/ingresses":
			data, err = os.ReadFile("fixtures/ing-list.json")
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
			// the type is present but there are no resources
			data = []byte(`{"items": []}`)
			require.NoError(t, err)
		case "/apis/awspca.cert-manager.io/v1beta1/awspcaissuers":
			data, err = os.ReadFile("fixtures/awspcaissuer-list.json")
			require.NoError(t, err)
		default:
			t.Fatalf("unexpected request: %s", r.URL.Path)
		}

		w.Write(data)
	}))

	cfg := &rest.Config{
		Host: server.URL,
	}

	status, err := GatherClusterStatus(context.Background(), cfg)
	require.NoError(t, err)

	assert.Equal(t, &ClusterStatus{
		Namespaces: []string{
			"jetstack-secure",
		},
		IngressShimIngresses: []summaryIngress{
			{
				Name:      "example",
				Namespace: "default",
				CertManagerAnnotations: map[string]string{
					"cert-manager.io/cluster-issuer": "nameOfClusterIssuer",
				},
			},
		},
		CRDGroups: []crdGroup{
			{
				Name: "cert-manager.io",
				CRDs: []string{
					"awspcaissuers.awspca.cert-manager.io",
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
					"googlecasclusterissuers.cas-issuer.jetstack.io",
					"googlecasissuers.cas-issuer.jetstack.io",
					"installations.operator.jetstack.io",
				},
			},
		},
		Components: map[string]installedComponent{
			"jetstack-secure-agent":        components.NewJetstackSecureAgentStatus("jetstack-secure", "v0.1.38"),
			"jetstack-secure-operator":     components.NewJetstackSecureOperatorStatus("jetstack-secure", "v0.0.1-alpha.17"),
			"cert-manager-controller":      components.NewCertManagerControllerStatus("jetstack-secure", "v1.9.1"),
			"cert-manager-cainjector":      components.NewCertManagerCAInjectorStatus("jetstack-secure", "v1.9.1"),
			"cert-manager-webhook":         components.NewCertManagerWebhookStatus("jetstack-secure", "v1.9.1"),
			"cert-manager-approver-policy": components.NewCertManagerApproverPolicyStatus("jetstack-secure", "v0.4.0"),
		},
		Issuers: []summaryIssuer{
			{
				Name:       "pca-sample",
				Namespace:  "jetstack-secure",
				Kind:       "AWSPCAIssuer",
				APIVersion: "awspca.cert-manager.io/v1beta1",
			},
			{
				Name:       "cm-cluster-issuer-sample",
				Namespace:  "",
				Kind:       "ClusterIssuer",
				APIVersion: "cert-manager.io/v1",
			},
			{
				Name:       "googlecasissuer-sample",
				Namespace:  "jetstack-secure",
				Kind:       "GoogleCASIssuer",
				APIVersion: "cas-issuer.jetstack.io/v1beta1",
			},
			{
				Name:       "cm-issuer-sample",
				Namespace:  "jetstack-secure",
				Kind:       "Issuer",
				APIVersion: "cert-manager.io/v1",
			},
		},
	}, status)
}
