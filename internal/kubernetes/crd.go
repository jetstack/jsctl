package kubernetes

import (
	"context"

	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"

	"github.com/jetstack/jsctl/internal/operator"
)

// CRDClient is used to query information on CRDs within a Kubernetes cluster.
type CRDClient struct {
	client *rest.RESTClient
}

// NewCRDClient returns a new instance of the CRDClient that can be used to
// query information on CRDs within a cluster
func NewCRDClient(config *rest.Config) (*CRDClient, error) {
	config.APIPath = "/apis"
	config.UserAgent = rest.DefaultKubernetesUserAgent()
	config.NegotiatedSerializer = serializer.NewCodecFactory(runtime.NewScheme())
	config.ContentConfig.GroupVersion = &schema.GroupVersion{
		Group:   v1.GroupName,
		Version: v1.SchemeGroupVersion.Version,
	}

	restClient, err := rest.UnversionedRESTClientFor(config)
	if err != nil {
		return nil, err
	}

	return &CRDClient{client: restClient}, nil
}

// InstallationStatus is a helper to see the status of the Installation CRD, the
// main installation used by the operator.
func (c *CRDClient) InstallationStatus(ctx context.Context) error {
	var err error

	err = c.client.Get().Resource("customresourcedefinitions").Name("installations.operator.jetstack.io").Do(ctx).Error()
	switch {
	case errors.IsNotFound(err):
		return operator.ErrNoInstallationCRD
	case err != nil:
		return err
	}

	return nil
}
