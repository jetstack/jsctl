package kubernetes

import (
	"context"
	"fmt"
	"sort"

	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
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

// Present returns true if the named CRD is present in the cluster.
func (c *CRDClient) Present(ctx context.Context, name string) (bool, error) {
	var err error

	err = c.client.Get().Resource("customresourcedefinitions").Name("installations.operator.jetstack.io").Do(ctx).Error()
	switch {
	case errors.IsNotFound(err):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("error querying for CRD: %w", err)
	}

	return true, nil
}

func (c *CRDClient) List(ctx context.Context) ([]string, error) {
	var crds v1.CustomResourceDefinitionList
	err := c.client.Get().Resource("customresourcedefinitions").Do(ctx).Into(&crds)
	if err != nil {
		return nil, fmt.Errorf("error querying for CRDs: %w", err)
	}

	var names []string
	for _, crd := range crds.Items {
		names = append(names, crd.Name)
	}

	sort.Strings(names)

	return names, nil
}
