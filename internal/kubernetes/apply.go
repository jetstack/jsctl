package kubernetes

import (
	"context"
	"fmt"
	"io"
	"os"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/restmapper"
)

type (
	// The StdOutApplier type applies YAML-encoded Kubernetes resources by writing them to os.Stdout.
	StdOutApplier struct{}
)

// NewStdOutApplier returns a new instance of the StdOutApplier type that will Apply YAML-encoded Kubernetes resources
// by writing them to os.Stdout
func NewStdOutApplier() *StdOutApplier {
	return &StdOutApplier{}
}

// Apply copies the content of r to os.Stdout.
func (s *StdOutApplier) Apply(_ context.Context, r io.Reader) error {
	_, err := io.Copy(os.Stdout, r)
	return err
}

type (
	// The KubeConfigApplier type applies YAML-encoded Kubernetes resources directly using the Kubernetes API.
	KubeConfigApplier struct {
		client dynamic.Interface
		mapper meta.RESTMapper
	}
)

// NewKubeConfigApplier returns a new instance of the KubeConfigApplier type that connects to a Kubernetes API server
// via the provided kubeconfig file location. If the provided location is blank, an in-cluster configuration is assumed.
func NewKubeConfigApplier(kubeConfig string) (*KubeConfigApplier, error) {
	config, err := NewConfig(kubeConfig)
	if err != nil {
		return nil, err
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	groupResources, err := restmapper.GetAPIGroupResources(clientSet.Discovery())
	if err != nil {
		return nil, err
	}

	mapper := restmapper.NewDiscoveryRESTMapper(groupResources)

	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &KubeConfigApplier{
		client: client,
		mapper: mapper,
	}, nil
}

// Apply the contents of the io.Reader implementation to the Kubernetes cluster described in the kubeconfig file. Any
// resources that already exist will be patched. It is assumed that the contents of the io.Reader implementation will
// be a YAML stream of Kubernetes resources separated by "---". The Apply operation can be cancelled via the provided
// context.Context.
func (k *KubeConfigApplier) Apply(ctx context.Context, r io.Reader) error {
	scanner := NewObjectScanner(r)
	const fieldManager = "kubectl-client-side-apply"

	return scanner.ForEach(ctx, func(ctx context.Context, object *unstructured.Unstructured) error {
		gvk := object.GroupVersionKind()
		mapping, err := k.mapper.RESTMapping(schema.GroupKind{Group: gvk.Group, Kind: gvk.Kind})
		if err != nil {
			return fmt.Errorf("error creating REST mapping for %s %s: %w", object.GetKind(), object.GetName(), err)
		}

		client := k.client.Resource(mapping.Resource).Namespace(object.GetNamespace())

		// TODO: output what resource is being created
		_, err = client.Create(ctx, object, metav1.CreateOptions{})
		if errors.IsAlreadyExists(err) {
			data, err := runtime.Encode(unstructured.UnstructuredJSONScheme, object)
			if err != nil {
				return fmt.Errorf("error encoding %s %s: %w", object.GetKind(), object.GetName(), err)
			}

			force := true

			// TODO: output what change is being applied
			_, err = client.Patch(ctx, object.GetName(), types.ApplyPatchType, data, metav1.PatchOptions{
				FieldManager: fieldManager,
				Force:        &force,
			})
			if err != nil {
				return fmt.Errorf("error applying patch update to %s %s: %w", object.GetKind(), object.GetName(), err)
			}
			return nil
		}
		if err != nil {
			return fmt.Errorf("error creating %s %s: %w", object.GetKind(), object.GetName(), err)
		}
		return nil
	})
}
