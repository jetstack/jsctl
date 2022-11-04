package clients

import (
	"context"
	"fmt"
	"sort"

	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/rest"
)

// CRDClient is used to query information on CRDs within a Kubernetes cluster.
type CRDClient struct {
	client *Generic[*v1.CustomResourceDefinition, *v1.CustomResourceDefinitionList]
}

// NewCRDClient returns a new instance of the CRDClient that can be used to
// query information on CRDs within a cluster
func NewCRDClient(config *rest.Config) (*CRDClient, error) {
	genericClient, err := NewGenericClient[*v1.CustomResourceDefinition, *v1.CustomResourceDefinitionList](
		config,
		v1.GroupName,
		v1.SchemeGroupVersion.Version,
		"customresourcedefinitions",
	)
	if err != nil {
		return nil, fmt.Errorf("error creating generic client: %w", err)
	}

	return &CRDClient{client: genericClient}, nil
}

// Present returns true if the named CRD is present in the cluster.
func (c *CRDClient) Present(ctx context.Context, name string) (bool, error) {

	var crd v1.CustomResourceDefinition

	err := c.client.Get(ctx, name, &crd)
	switch {
	case errors.IsNotFound(err):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("error querying for CRD: %w", err)
	}

	return true, nil
}

func (c *CRDClient) List(ctx context.Context) ([]string, error) {
	var crdList v1.CustomResourceDefinitionList

	err := c.client.List(ctx, &crdList)
	if err != nil {
		return nil, fmt.Errorf("error listing CRDs: %w", err)
	}

	var names []string
	for _, crd := range crdList.Items {
		names = append(names, crd.Name)
	}

	sort.Strings(names)

	return names, nil
}
