package status

import (
	"context"
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	v1networking "k8s.io/api/networking/v1"
	v1extensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/client-go/rest"

	"github.com/jetstack/jsctl/internal/kubernetes/clients"
)

// ClusterPreInstallStatus is a collection of information about a cluster that
// can be helpful for users about to install.
type ClusterPreInstallStatus struct {
	CRDGroups []crdGroup `yaml:"crds"`
	// Namespaces is a list of namespaces that exist in the cluster which are
	// related to Jetstack Secure components
	Namepaces []string `yaml:"namespaces"`
	// Ingresses is a list of ingresses in the cluster related to cert-manager
	Ingresses []summaryIngress `yaml:"ingresses"`
}

// crdGroup is a list of custom resource definitions that are all part of the
// same group, e.g. cert-manager.io or jetstack.io.
type crdGroup struct {
	Name string
	CRDs []string `yaml:"items"`
}

// summaryIngress is a wrapper of some summary information about an ingress
// related to cert-manager.
type summaryIngress struct {
	Name                   string            `yaml:"name"`
	Namespace              string            `yaml:"namespace"`
	CertManagerAnnotations map[string]string `yaml:"certManagerAnnotations"`
}

// GatherClusterPreInstallStatus returns a ClusterPreInstallStatus for the
// supplied cluster
func GatherClusterPreInstallStatus(ctx context.Context, cfg *rest.Config) (*ClusterPreInstallStatus, error) {
	var err error
	var status ClusterPreInstallStatus

	// gather the namespaces in the cluster and list only the ones related to
	// Jetstack Secure
	namespaceClient, err := clients.NewGenericClient[*v1.Namespace, *v1.NamespaceList](
		&clients.GenericClientOptions{
			RestConfig: cfg,
			APIPath:    "/api/",
			Group:      v1.GroupName,
			Version:    v1.SchemeGroupVersion.Version,
			Kind:       "namespaces",
		},
	)

	var namespaces v1.NamespaceList
	err = namespaceClient.List(ctx, &clients.GenericRequestOptions{}, &namespaces)
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %s", err)
	}

	for _, namespace := range namespaces.Items {
		if namespace.Name == "cert-manager" || namespace.Name == "jetstack-secure" {
			status.Namepaces = append(status.Namepaces, namespace.Name)
		}
	}

	// gather the crds present in the cluster
	crdClient, err := clients.NewCRDClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("error creating CRD client: %w", err)
	}

	groups := []string{
		"cert-manager.io",
		"jetstack.io",
	}

	var crdList v1extensions.CustomResourceDefinitionList

	err = crdClient.List(ctx, &clients.GenericRequestOptions{}, &crdList)
	if err != nil {
		return nil, fmt.Errorf("error querying for CRDs: %w", err)
	}

	for _, g := range groups {
		var crdGroup crdGroup
		crdGroup.Name = g
		for _, crd := range crdList.Items {
			if strings.HasSuffix(crd.Name, g) {
				crdGroup.CRDs = append(crdGroup.CRDs, crd.Name)
			}
		}
		status.CRDGroups = append(status.CRDGroups, crdGroup)
	}

	// gather ingresses related to cert-manager in the cluster
	ingressClient, err := clients.NewGenericClient[*v1networking.Ingress, *v1networking.IngressList](
		&clients.GenericClientOptions{
			RestConfig: cfg,
			Group:      v1networking.GroupName,
			Version:    v1networking.SchemeGroupVersion.Version,
			Kind:       "ingresses",
		},
	)

	var ingresses v1networking.IngressList
	err = ingressClient.List(ctx, &clients.GenericRequestOptions{}, &ingresses)
	if err != nil {
		return nil, fmt.Errorf("failed to list ingresses: %s", err)
	}

	for _, ingress := range ingresses.Items {
		relatedToCertManager := false
		for k := range ingress.Annotations {
			if strings.HasPrefix(k, "cert-manager.io") {
				relatedToCertManager = true
				break
			}
		}
		if !relatedToCertManager {
			continue
		}
		status.Ingresses = append(status.Ingresses, summaryIngress{
			Name:      ingress.Name,
			Namespace: ingress.Namespace,
			CertManagerAnnotations: func() map[string]string {
				selectedAnnotations := make(map[string]string)
				for k, v := range ingress.Annotations {
					if strings.HasPrefix(k, "cert-manager.io") {
						selectedAnnotations[k] = v
					}
				}
				return selectedAnnotations
			}(),
		})
	}

	return &status, nil
}
