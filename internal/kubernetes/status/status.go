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
	"github.com/jetstack/jsctl/internal/kubernetes/status/components"
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

	// Components is a list of components installed in the cluster which are
	// cert-manager or jetstack-secure related
	Components map[string]installedComponent `yaml:"components"`
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

// installedComponent is a interface which a custom component status must
// implement. This is designed to be extended to support other components with
// more interesting statuses in the future while supporting the base ones too.
type installedComponent interface {
	Name() string
	Namespace() string
	Version() string
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

	// gather pods and identify the relevant installted components
	podClient, err := clients.NewGenericClient[*v1.Pod, *v1.PodList](
		&clients.GenericClientOptions{
			RestConfig: cfg,
			APIPath:    "/api/",
			Group:      v1.GroupName,
			Version:    v1.SchemeGroupVersion.Version,
			Kind:       "pods",
		},
	)

	var pods v1.PodList
	err = podClient.List(ctx, &clients.GenericRequestOptions{}, &pods)
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %s", err)
	}

	componentStatuses, err := findComponents(pods.Items)
	if err != nil {
		return nil, fmt.Errorf("failed to identify components in the cluster: %s", err)
	}

	status.Components = componentStatuses

	return &status, nil
}

func findComponents(pods []v1.Pod) (map[string]installedComponent, error) {
	componentStatuses := make(map[string]installedComponent)

	for _, pod := range pods {
		certManagerControllerStatus, err := components.FindCertManagerController(&pod)
		if err != nil {
			return nil, fmt.Errorf("failed while testing pod as cert-manager-controller: %s", err)
		}
		if certManagerControllerStatus != nil {
			componentStatuses[certManagerControllerStatus.Name()] = certManagerControllerStatus
		}

		jetstackSecureAgentStatus, err := components.FindJetstackSecureAgent(&pod)
		if err != nil {
			return nil, fmt.Errorf("failed while testing pod as jetstack-secure-agent: %s", err)
		}
		if jetstackSecureAgentStatus != nil {
			componentStatuses[jetstackSecureAgentStatus.Name()] = jetstackSecureAgentStatus
		}
	}

	return componentStatuses, nil
}
