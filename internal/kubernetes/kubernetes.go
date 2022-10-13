// Package kubernetes provides types and methods for communicating with Kubernetes clusters and resources.
package kubernetes

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/mitchellh/go-homedir"
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
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"
)

// NewConfig returns a new rest.Config instance based on the kubeconfig path provided. If the path is blank, an in-cluster
// configuration is assumed.
func NewConfig(kubeConfig string) (*rest.Config, error) {
	var config *rest.Config
	var err error
	if kubeConfig != "" {
		kubeConfigPath, err := homedir.Expand(kubeConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to expand kubeconfig path: %w", err)
		}

		_, err = os.Stat(kubeConfigPath)
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("kubeconfig doesn't exist: %w", err)
		} else if err != nil {
			return nil, fmt.Errorf("failed to check kubeconfig path: %w", err)
		}

		config, err = clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	} else {
		config, err = rest.InClusterConfig()
	}

	if err != nil {
		return nil, err
	}

	if config == nil {
		return nil, fmt.Errorf("failed to create config, is your kubeconfig present and configured to connect to a cluster that's still running?")
	}

	return config, nil
}

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
			return err
		}

		client := k.client.Resource(mapping.Resource).Namespace(object.GetNamespace())

		_, err = client.Create(ctx, object, metav1.CreateOptions{})
		if errors.IsAlreadyExists(err) {
			data, err := runtime.Encode(unstructured.UnstructuredJSONScheme, object)
			if err != nil {
				return err
			}

			force := true

			_, err = client.Patch(ctx, object.GetName(), types.ApplyPatchType, data, metav1.PatchOptions{
				FieldManager: fieldManager,
				Force:        &force,
			})
			return err
		}

		return err
	})
}

type (
	// The ObjectScanner type is used to parse a YAML stream of Kubernetes resources and invoke a callback for each one.
	ObjectScanner struct {
		reader io.Reader
	}

	// The ObjectCallback type is a function that is invoked for each Kubernetes object parsed when calling
	// ObjectScanner.Apply.
	ObjectCallback func(ctx context.Context, object *unstructured.Unstructured) error
)

// NewObjectScanner returns a new instance of the ObjectScanner type that will parse the provided io.Reader's data
// as a YAML-encoded stream of Kubernetes resources.
func NewObjectScanner(r io.Reader) *ObjectScanner {
	return &ObjectScanner{reader: r}
}

// ForEach iterates through the stream of YAML-encoded Kubernetes resources and invokes the ObjectCallback for each
// one. Iteration can be cancelled by the ObjectCallback returning a non-nil error or by cancelling the provided
// context.Context.
func (oj *ObjectScanner) ForEach(ctx context.Context, fn ObjectCallback) error {
	const separator = "---"

	scanner := bufio.NewScanner(oj.reader)

	buf := bytes.NewBuffer([]byte{})
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			scanner.Scan()
			line := scanner.Bytes()

			switch {
			case line == nil && buf.Len() == 0:
				return nil
			case line == nil && buf.Len() > 0:
				break
			case string(line) != separator:
				buf.Write(line)
				buf.WriteRune('\n')
				continue
			}

			var object unstructured.Unstructured
			if err := yaml.Unmarshal(buf.Bytes(), &object); err != nil {
				return err
			}

			buf.Reset()
			if err := fn(ctx, &object); err != nil {
				return err
			}
		}
	}
}
