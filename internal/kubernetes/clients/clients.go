package clients

import (
	"fmt"

	v1extensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/client-go/rest"
)

// NewCRDClient returns an instance of a generic client for querying CRDs
func NewCRDClient(config *rest.Config) (*Generic[*v1extensions.CustomResourceDefinition, *v1extensions.CustomResourceDefinitionList], error) {
	genericClient, err := NewGenericClient[*v1extensions.CustomResourceDefinition, *v1extensions.CustomResourceDefinitionList](
		&GenericClientOptions{
			RestConfig: config,
			APIPath:    "/apis",
			Group:      v1extensions.GroupName,
			Version:    v1extensions.SchemeGroupVersion.Version,
			Kind:       "customresourcedefinitions",
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error creating generic client: %w", err)
	}

	return genericClient, nil
}
