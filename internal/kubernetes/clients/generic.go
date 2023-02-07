package clients

import (
	"context"
	"fmt"

	"github.com/Jeffail/gabs/v2"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/rest"
)

type Generic[T, ListT runtime.Object] interface {
	Get(context.Context, *GenericRequestOptions, T) error
	List(context.Context, *GenericRequestOptions, ListT) error
	Present(ctx context.Context, options *GenericRequestOptions) (bool, error)
	Patch(ctx context.Context, options *GenericRequestOptions, patch []byte) error
}

type generic[T, ListT runtime.Object] struct {
	restClient rest.Interface
	resource   string
}

var _ Generic[*runtime.Unknown, *runtime.Unknown] = &generic[*runtime.Unknown, *runtime.Unknown]{}

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

	// DropFields is a list of fields to drop from the response
	DropFields []string
}

// NewGenericClient returns a new instance of a Generic client configured to
// query the specified resource. Use type parameters for types and list types
// for the desired result types and gvk function parameters to specify the
// group, version, and kind of the resource to query.
func NewGenericClient[T, ListT runtime.Object](opts *GenericClientOptions) (Generic[T, ListT], error) {
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

	return &generic[T, ListT]{
		restClient: restClient,
		resource:   opts.Kind,
	}, nil
}

func (c *generic[T, ListT]) Get(ctx context.Context, options *GenericRequestOptions, result T) error {
	r := c.restClient.Get().Resource(c.resource)

	if options.Namespace != "" {
		r = r.Namespace(options.Namespace)
	}
	if options.Name != "" {
		r = r.Name(options.Name)
	}

	// DoRaw allows us to mutate the response if we're dropping fields
	jsonBody, err := r.DoRaw(ctx)
	if err != nil {
		return fmt.Errorf("error getting %T: %w", result, err)
	}

	if len(options.DropFields) > 0 {
		// parse the JSON for processing in gabs
		container, err := gabs.ParseJSON(jsonBody)
		if err != nil {
			return fmt.Errorf("failed to parse generated json for resource: %s", err)
		}

		// craft a new object containing only selected fields
		for _, v := range options.DropFields {
			// also support JSONPointers for keys containing '.' chars
			pathComponents, err := gabs.JSONPointerToSlice(v)
			if err != nil {
				return fmt.Errorf("invalid JSONPointer: %s", v)
			}
			if container.Exists(pathComponents...) {
				err := container.Delete(pathComponents...)
				if err != nil {
					return fmt.Errorf("failed to delete field: %s", err)
				}
			}
		}

		jsonBody = container.Bytes()
	}

	err = json.Unmarshal(jsonBody, result)
	if err != nil {
		return fmt.Errorf("failed to unmarshal resource: %s", err)
	}

	return nil
}

// List is must the same as get, however it returns results in a list type
// instead
func (c *generic[T, ListT]) List(ctx context.Context, options *GenericRequestOptions, result ListT) error {
	r := c.restClient.Get().Resource(c.resource)

	if options.Namespace != "" {
		r = r.Namespace(options.Namespace)
	}

	jsonBody, err := r.DoRaw(ctx)
	if err != nil {
		return fmt.Errorf("error listing %T: %w", result, err)
	}

	if len(options.DropFields) > 0 {

		// parse the JSON for processing in gabs
		container, err := gabs.ParseJSON(jsonBody)
		if err != nil {
			return fmt.Errorf("failed to parse generated json for resource: %s", err)
		}

		var items []interface{}
		for _, i := range container.Search("items").Children() {
			for _, v := range options.DropFields {
				// also support JSONPointers for keys containing '.' chars
				pathComponents, err := gabs.JSONPointerToSlice(v)
				if err != nil {
					return fmt.Errorf("invalid JSONPointer: %s", v)
				}
				if i.Exists(pathComponents...) {
					err := i.Delete(pathComponents...)
					if err != nil {
						return fmt.Errorf("failed to delete field: %s", err)
					}
				}
			}
			items = append(items, i.Data())
		}

		_, err = container.Set(items, "items")
		if err != nil {
			return fmt.Errorf("failed to update filtered items: %s", err)
		}

		jsonBody = container.Bytes()
	}

	err = json.Unmarshal(jsonBody, result)
	if err != nil {
		return fmt.Errorf("failed to unmarshal resource: %s", err)
	}

	return nil
}

func (c *generic[T, ListT]) Present(ctx context.Context, options *GenericRequestOptions) (bool, error) {
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

func (c *generic[T, ListT]) Patch(ctx context.Context, options *GenericRequestOptions, patch []byte) error {
	r := c.restClient.Patch(types.StrategicMergePatchType).Body(patch).Resource(c.resource)

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
