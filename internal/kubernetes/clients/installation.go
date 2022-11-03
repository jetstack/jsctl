package clients

import (
	"context"
	"errors"
	"sort"

	"github.com/jetstack/js-operator/pkg/apis/operator/v1alpha1"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
)

type (
	// The InstallationClient is used to query information on an Installation resource within a Kubernetes cluster.
	InstallationClient struct {
		client *rest.RESTClient
	}

	// ComponentStatus describes the status of an individual operator component.
	ComponentStatus struct {
		Name    string `json:"name"`
		Ready   bool   `json:"ready"`
		Message string `json:"message,omitempty"`
	}
)

// NewInstallationClient returns a new instance of the InstallationClient that will interact with the Kubernetes
// cluster specified in the rest.Config.
func NewInstallationClient(config *rest.Config) (*InstallationClient, error) {
	// Set up the rest config to obtain Installation resources
	config.APIPath = "/apis"
	config.UserAgent = rest.DefaultKubernetesUserAgent()
	config.NegotiatedSerializer = serializer.NewCodecFactory(v1alpha1.GlobalScheme)
	config.ContentConfig.GroupVersion = &schema.GroupVersion{
		Group:   v1alpha1.InstallationGVK.Group,
		Version: v1alpha1.InstallationGVK.Version,
	}

	restClient, err := rest.UnversionedRESTClientFor(config)
	if err != nil {
		return nil, err
	}

	return &InstallationClient{client: restClient}, nil
}

var (
	// ErrNoInstallation is the error given when querying an Installation resource that does not exist.
	ErrNoInstallation = errors.New("no installation")

	// ErrNoInstallationCRD is the error given when the Installation CRD does not exist in the cluster.
	ErrNoInstallationCRD = errors.New("no installation CRD")

	componentNames = map[v1alpha1.InstallationConditionType]string{
		v1alpha1.InstallationConditionCertManagerReady:        "cert-manager",
		v1alpha1.InstallationConditionCertManagerIssuersReady: "issuers",
		v1alpha1.InstallationConditionCSIDriversReady:         "csi-driver",
		v1alpha1.InstallationConditionIstioCSRReady:           "istio-csr",
		v1alpha1.InstallationConditionApproverPolicyReady:     "approver-policy",
		v1alpha1.InstallationConditionVenafiOauthHelperReady:  "venafi-oauth-helper",
		v1alpha1.InstallationConditionManifestsReady:          "manifests",
	}
)

// Status returns a slice of ComponentStatus types that describe the state of individual components installed by the
// operator. Returns ErrNoInstallation if an Installation resource cannot be found in the cluster. It uses the
// status conditions on an Installation resource and maps those to a ComponentStatus, the ComponentStatus.Name field
// is chosen based on the content of the componentNames map. Add friendly names to that map to include additional
// component statuses to return.
func (ic *InstallationClient) Status(ctx context.Context) ([]ComponentStatus, error) {
	var err error
	var installation v1alpha1.Installation

	const (
		resource = "installations"
		name     = "installation"
	)

	err = ic.client.Get().Resource(resource).Name(name).Do(ctx).Into(&installation)
	switch {
	case errors2.IsNotFound(err):
		return nil, ErrNoInstallation
	case err != nil:
		return nil, err
	}

	statuses := make([]ComponentStatus, 0)
	for _, condition := range installation.Status.Conditions {
		componentStatus := ComponentStatus{
			Ready: condition.Status == v1alpha1.ConditionTrue,
		}

		// Don't place the message if the component is considered ready.
		if !componentStatus.Ready {
			componentStatus.Message = condition.Message
		}

		// Swap the condition type for its friendly component name, don't include anything we don't have
		// a friendly name for.
		componentName, ok := componentNames[condition.Type]
		if !ok {
			continue
		}

		componentStatus.Name = componentName
		statuses = append(statuses, componentStatus)
	}

	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].Name < statuses[j].Name
	})

	return statuses, nil
}
