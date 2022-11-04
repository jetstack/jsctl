package clients

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
)

// Generic is a client which can be configured to query any Kubernetes resource
type Generic[T, ListT runtime.Object] struct {
	restClient *rest.RESTClient
	resource   string
}

// NewGenericClient returns a new instance of a Generic client configured to
// query the specified resource. Use type parameters for types and list types
// for the desired result types and gvk function parameters to specify the
// group, version, and kind of the resource to query.
func NewGenericClient[T, ListT runtime.Object](config *rest.Config, group, version, kind string) (*Generic[T, ListT], error) {
	config.APIPath = "/apis"
	config.UserAgent = rest.DefaultKubernetesUserAgent()
	config.NegotiatedSerializer = serializer.NewCodecFactory(runtime.NewScheme())
	config.ContentConfig.GroupVersion = &schema.GroupVersion{
		Group:   group,
		Version: version,
	}

	restClient, err := rest.UnversionedRESTClientFor(config)
	if err != nil {
		return nil, err
	}

	return &Generic[T, ListT]{
		restClient: restClient,
		resource:   kind,
	}, nil
}

func (c *Generic[T, ListT]) Get(ctx context.Context, name string, result T) error {
	err := c.restClient.Get().
		Resource(c.resource).
		Name(name).
		Do(ctx).
		Into(result)
	if err != nil {
		return fmt.Errorf("error getting %T: %w", result, err)
	}

	return nil
}

func (c *Generic[T, ListT]) List(ctx context.Context, result ListT) error {
	err := c.restClient.
		Get().
		Resource(c.resource).
		Do(ctx).
		Into(result)
	if err != nil {
		return fmt.Errorf("error listing %T: %w", result, err)
	}

	return nil
}
