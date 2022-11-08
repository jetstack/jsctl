package status

import (
	"context"
	"fmt"
	"strings"

	awspca "github.com/cert-manager/aws-privateca-issuer/pkg/api/v1beta1"
	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	googlecas "github.com/jetstack/google-cas-issuer/api/v1beta1"
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

	// Issuers is a list of issuers of all kinds found in the cluster. Including
	// external issuers.
	Issuers []summaryIssuer `yaml:"issuers"`
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

// summaryIssuer is a wrapper of some summary information about an issuer
type summaryIssuer struct {
	Name      string `yaml:"name"`
	Kind      string `yaml:"kind"`
	Namespace string `yaml:"namespace,omitempty"`
}

// installedComponent is a interface which a custom component status must
// implement. This is designed to be extended to support other components with
// more interesting statuses in the future while supporting the base ones too.
type installedComponent interface {
	Name() string
	Namespace() string
	Version() string

	// Match will populate the installedComponent with information from the pod
	// if the pod is determined to be a pod from that component
	Match(pod *v1.Pod) (bool, error)
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

	// gather pods and identify the relevant installed components
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

	status.Components, err = findComponents(pods.Items)
	if err != nil {
		return nil, fmt.Errorf("failed to identify components in the cluster: %s", err)
	}

	// gather issuers and find each issuer of each kind
	status.Issuers, err = findIssuers(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed while finding issuers in the cluster: %s", err)
	}

	return &status, nil
}

func findIssuers(ctx context.Context, cfg *rest.Config) ([]summaryIssuer, error) {
	issuerClient, err := clients.NewAllIssuers(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create issuer client: %s", err)
	}
	issuerKinds, err := issuerClient.ListKinds(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list issuer kinds: %s", err)
	}

	var summaryIssuers []summaryIssuer
	for _, kind := range issuerKinds {
		switch kind {
		case clients.CertManagerIssuer:
			client, err := clients.NewCertManagerIssuerClient(cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to create clusterissuer client: %s", err)
			}
			var issuers cmapi.IssuerList
			err = client.List(ctx, &clients.GenericRequestOptions{}, &issuers)
			if err != nil {
				return nil, fmt.Errorf("failed to list clusterissuers: %s", err)
			}
			for _, issuer := range issuers.Items {
				summaryIssuers = append(summaryIssuers, summaryIssuer{
					Name:      issuer.Name,
					Namespace: issuer.Namespace,
					Kind:      issuer.Kind,
				})
			}
		case clients.CertManagerClusterIssuer:
			client, err := clients.NewCertManagerClusterIssuerClient(cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to create clusterissuer client: %s", err)
			}
			var clusterIssuers cmapi.ClusterIssuerList
			err = client.List(ctx, &clients.GenericRequestOptions{}, &clusterIssuers)
			if err != nil {
				return nil, fmt.Errorf("failed to list clusterissuers: %s", err)
			}
			for _, issuer := range clusterIssuers.Items {
				summaryIssuers = append(summaryIssuers, summaryIssuer{
					Name:      issuer.Name,
					Namespace: issuer.Namespace,
					Kind:      issuer.Kind,
				})
			}
		case clients.GoogleCASIssuer:
			client, err := clients.NewGoogleCASIssuerClient(cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to create cas client: %s", err)
			}
			var issuers googlecas.GoogleCASIssuerList
			err = client.List(ctx, &clients.GenericRequestOptions{}, &issuers)
			if err != nil {
				return nil, fmt.Errorf("failed to list cas issuers: %s", err)
			}
			for _, issuer := range issuers.Items {
				summaryIssuers = append(summaryIssuers, summaryIssuer{
					Name:      issuer.Name,
					Namespace: issuer.Namespace,
					Kind:      issuer.Kind,
				})
			}
		case clients.GoogleCASClusterIssuer:
			client, err := clients.NewGoogleCASClusterIssuerClient(cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to create cas cluster issuer client: %s", err)
			}
			var issuers googlecas.GoogleCASClusterIssuerList
			err = client.List(ctx, &clients.GenericRequestOptions{}, &issuers)
			if err != nil {
				return nil, fmt.Errorf("failed to list cas cluster issuers: %s", err)
			}
			for _, issuer := range issuers.Items {
				summaryIssuers = append(summaryIssuers, summaryIssuer{
					Name:      issuer.Name,
					Namespace: issuer.Namespace,
					Kind:      issuer.Kind,
				})
			}
		case clients.AWSPCAIssuer:
			client, err := clients.NewAWSPCAIssuerClient(cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to create aws pca issuer client: %s", err)
			}
			var issuers awspca.AWSPCAIssuerList
			err = client.List(ctx, &clients.GenericRequestOptions{}, &issuers)
			if err != nil {
				return nil, fmt.Errorf("failed to list pca issuers: %s", err)
			}
			for _, issuer := range issuers.Items {
				summaryIssuers = append(summaryIssuers, summaryIssuer{
					Name:      issuer.Name,
					Namespace: issuer.Namespace,
					Kind:      issuer.Kind,
				})
			}
		case clients.AWSPCAClusterIssuer:
			client, err := clients.NewAWSPCAClusterIssuerClient(cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to create aws pca cluster issuer client: %s", err)
			}
			var issuers awspca.AWSPCAClusterIssuerList
			err = client.List(ctx, &clients.GenericRequestOptions{}, &issuers)
			if err != nil {
				return nil, fmt.Errorf("failed to list pca cluster issuers: %s", err)
			}
			for _, issuer := range issuers.Items {
				summaryIssuers = append(summaryIssuers, summaryIssuer{
					Name:      issuer.Name,
					Namespace: issuer.Namespace,
					Kind:      issuer.Kind,
				})
			}
		}
	}

	return summaryIssuers, nil
}

// findComponents takes a list of pods and returns a list of detected components
// which might be relevant to Jetstack Secure
func findComponents(pods []v1.Pod) (map[string]installedComponent, error) {
	foundComponents := make(map[string]installedComponent)

	knownComponents := []installedComponent{
		&components.CertManagerControllerStatus{},
		&components.CertManagerCAInjectorStatus{},
		&components.CertManagerWebhookStatus{},

		&components.CertManagerCSIDriverStatus{},

		&components.CertManagerCSIDriverSPIFFEStatus{},
		&components.CertManagerCSIDriverSpiffeApproverStatus{},

		&components.CertManagerApproverPolicyStatus{},
		&components.CertManagerApproverPolicyEnterpriseStatus{},

		&components.JetstackSecureAgentStatus{},
		&components.JetstackSecureOperatorStatus{},

		&components.VenafiOAuthHelperStatus{},
		&components.CertDiscoveryVenafiStatus{},

		&components.GoogleCASIssuerStatus{},
		&components.AWSPCAIssuerStatus{},
		&components.KMSIssuerStatus{},
		&components.OriginCAIssuerStatus{},
		&components.SmallStepIssuerStatus{},
	}

	for i := range knownComponents {
		component := knownComponents[i]
		var found bool
		var err error
		for _, pod := range pods {
			found, err = component.Match(&pod)
			if err != nil {
				return nil, fmt.Errorf("failed while testing pod as %s: %s", component.Name(), err)
			}
			if found {
				break
			}
		}
		if found {
			foundComponents[component.Name()] = component
		}
	}

	return foundComponents, nil
}
