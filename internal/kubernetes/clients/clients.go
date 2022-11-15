package clients

import (
	"fmt"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/client-go/rest"
)

// NewCRDClient returns an instance of a generic client for querying CRDs
func NewCRDClient(config *rest.Config) (*Generic[*apiextensionsv1.CustomResourceDefinition, *apiextensionsv1.CustomResourceDefinitionList], error) {
	genericClient, err := NewGenericClient[*apiextensionsv1.CustomResourceDefinition, *apiextensionsv1.CustomResourceDefinitionList](
		&GenericClientOptions{
			RestConfig: config,
			APIPath:    "/apis",
			Group:      apiextensionsv1.GroupName,
			Version:    apiextensionsv1.SchemeGroupVersion.Version,
			Kind:       "customresourcedefinitions",
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error creating generic client: %w", err)
	}

	return genericClient, nil
}
