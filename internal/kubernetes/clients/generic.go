package clients

import (
	"context"
	"fmt"

	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
)

// Generic is a client which can be configured to query any Kubernetes resource
type Generic[T, ListT runtime.Object] struct {
	restClient *rest.RESTClient
	resource   string
}

// GenericClientOptions wrap the options for a Generic client initialization
type GenericClientOptions struct {
	RestConfig *rest.Config
	APIPath    string
	Group      string
	Version    string
	Kind       string
}

// GenericRequestOptions wrap data used to form requests
type GenericRequestOptions struct {
	// Name is the name of the resource to fetch. Set only when fetching a single
	// resource.
	Name string
	// Namespace is the name of the namespace to fetch resources from. Set only
	// when fetching resources in a single namespace and namespaced resources.
	Namespace string
}

// NewGenericClient returns a new instance of a Generic client configured to
// query the specified resource. Use type parameters for types and list types
// for the desired result types and gvk function parameters to specify the
// group, version, and kind of the resource to query.
func NewGenericClient[T, ListT runtime.Object](opts *GenericClientOptions) (*Generic[T, ListT], error) {
	config := opts.RestConfig
	config.UserAgent = rest.DefaultKubernetesUserAgent()
	config.NegotiatedSerializer = serializer.NewCodecFactory(runtime.NewScheme())
	config.ContentConfig.GroupVersion = &schema.GroupVersion{
		Group:   opts.Group,
		Version: opts.Version,
	}

	// v1 apis are available at a different path
	if opts.APIPath != "" {
		config.APIPath = opts.APIPath
	} else {
		config.APIPath = "/apis/"
	}

	restClient, err := rest.UnversionedRESTClientFor(config)
	if err != nil {
		return nil, err
	}

	return &Generic[T, ListT]{
		restClient: restClient,
		resource:   opts.Kind,
	}, nil
}

func (c *Generic[T, ListT]) Get(ctx context.Context, options *GenericRequestOptions, result T) error {
	r := c.restClient.Get().Resource(c.resource)

	if options.Namespace != "" {
		r = r.Namespace(options.Namespace)
	}
	if options.Name != "" {
		r = r.Name(options.Name)
	}

	err := r.Do(ctx).Into(result)
	if err != nil {
		return fmt.Errorf("error getting %T: %w", result, err)
	}

	return nil
}

// List is must the same as get, however it returns results in a list type
// instead
func (c *Generic[T, ListT]) List(ctx context.Context, options *GenericRequestOptions, result ListT) error {
	r := c.restClient.Get().Resource(c.resource)

	if options.Namespace != "" {
		r = r.Namespace(options.Namespace)
	}

	err := r.Do(ctx).Into(result)
	if err != nil {
		return fmt.Errorf("error listing %T: %w", result, err)
	}

	return nil
}

func (c *Generic[T, ListT]) Present(ctx context.Context, options *GenericRequestOptions) (bool, error) {
	r := c.restClient.Get().Resource(c.resource)

	if options.Namespace != "" {
		r = r.Namespace(options.Namespace)
	}
	if options.Name != "" {
		r = r.Name(options.Name)
	}

	err := r.Do(ctx).Error()
	switch {
	case apiErrors.IsNotFound(err):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("error testing presence: %w", err)
	}

	return true, nil
}

func (c *Generic[T, ListT]) Patch(ctx context.Context, options *GenericRequestOptions, patch []byte) error {
	r := c.restClient.Patch(types.MergePatchType).Body(patch).Resource(c.resource)

	if options.Namespace != "" {
		r = r.Namespace(options.Namespace)
	}
	if options.Name != "" {
		r = r.Name(options.Name)
	}

	err := r.Do(ctx).Error()
	if err != nil {
		return fmt.Errorf("error patching resource: %w", err)
	}

	return nil
}
