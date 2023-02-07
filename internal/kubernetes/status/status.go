package status

import (
	"context"
	"fmt"
	"strings"

	kmsissuerv1alpha1 "github.com/Skyscanner/kms-issuer/apis/certmanager/v1alpha1"
	awspcaissuerv1beta1 "github.com/cert-manager/aws-privateca-issuer/pkg/api/v1beta1"
	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	origincaissuerv1 "github.com/cloudflare/origin-ca-issuer/pkgs/apis/v1"
	googlecasissuerv1beta1 "github.com/jetstack/google-cas-issuer/api/v1beta1"
	veiv1alpha1 "github.com/jetstack/venafi-enhanced-issuer/api/v1alpha1"
	stepissuerv1beta1 "github.com/smallstep/step-issuer/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/client-go/rest"

	"github.com/jetstack/jsctl/internal/kubernetes/clients"
	"github.com/jetstack/jsctl/internal/kubernetes/status/components"
)

// ClusterStatus is a collection of information about a cluster that
// can be helpful for users about to install.
type ClusterStatus struct {
	// CRDGroups is a series of groups of CRDs by their domain, e.g. jetstack.io
	CRDGroups []crdGroup `yaml:"crds"`

	// Namespaces is a list of namespaces that exist in the cluster which are
	// related to Jetstack Secure components
	Namespaces []string `yaml:"namespaces"`

	// IngressShimIngresses is a list of ingresses in the cluster using cert-manager ingress shim
	IngressShimIngresses []summaryIngress `yaml:"ingress-shim-ingresses"`

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
	// APIVersion is the API group name and the version
	APIVersion string `yaml:"apiVersion"`

	// Kind is the name of the kind in that API group
	Kind string `yaml:"kind"`

	// Name is the name of that Issuer resource
	Name string `yaml:"name"`

	// Namespace is the namespace of that Issuer resource if the Issuer is not
	// cluster scoped
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
	Match(md *components.MatchData) (bool, error)
}

// GatherClusterStatus returns a ClusterStatus for the
// supplied cluster
func GatherClusterStatus(ctx context.Context, cfg *rest.Config) (*ClusterStatus, error) {
	var err error
	var status ClusterStatus

	// gather the namespaces in the cluster and list only the ones related to
	// Jetstack Secure
	namespaceClient, err := clients.NewGenericClient[*corev1.Namespace, *corev1.NamespaceList](
		&clients.GenericClientOptions{
			RestConfig: cfg,
			APIPath:    "/api/",
			Group:      corev1.GroupName,
			Version:    corev1.SchemeGroupVersion.Version,
			Kind:       "namespaces",
		},
	)

	var namespaces corev1.NamespaceList
	err = namespaceClient.List(ctx, &clients.GenericRequestOptions{}, &namespaces)
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %s", err)
	}

	for _, namespace := range namespaces.Items {
		if namespace.Name == "cert-manager" || namespace.Name == "jetstack-secure" {
			status.Namespaces = append(status.Namespaces, namespace.Name)
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

	var crdList apiextensionsv1.CustomResourceDefinitionList

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

	// gather ingress shim ingresses, based on their annotations
	ingressClient, err := clients.NewGenericClient[*networkingv1.Ingress, *networkingv1.IngressList](
		&clients.GenericClientOptions{
			RestConfig: cfg,
			Group:      networkingv1.GroupName,
			Version:    networkingv1.SchemeGroupVersion.Version,
			Kind:       "ingresses",
		},
	)

	var ingresses networkingv1.IngressList
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
			// kube-lego annotatio
			if k == "kubernetes.io/tls-acme" {
				relatedToCertManager = true
				break
			}
		}
		if !relatedToCertManager {
			continue
		}
		status.IngressShimIngresses = append(status.IngressShimIngresses, summaryIngress{
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
	podClient, err := clients.NewGenericClient[*corev1.Pod, *corev1.PodList](
		&clients.GenericClientOptions{
			RestConfig: cfg,
			APIPath:    "/api/",
			Group:      corev1.GroupName,
			Version:    corev1.SchemeGroupVersion.Version,
			Kind:       "pods",
		},
	)

	var pods corev1.PodList
	err = podClient.List(ctx, &clients.GenericRequestOptions{}, &pods)
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %s", err)
	}

	md := components.MatchData{
		Pods: pods.Items,
	}

	status.Components, err = findComponents(&md)
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
					APIVersion: cmapi.SchemeGroupVersion.String(),
					Name:       issuer.Name,
					Namespace:  issuer.Namespace,
					Kind:       issuer.Kind,
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
					APIVersion: cmapi.SchemeGroupVersion.String(),
					Name:       issuer.Name,
					Kind:       issuer.Kind,
				})
			}
		case clients.GoogleCASIssuer:
			client, err := clients.NewGoogleCASIssuerClient(cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to create cas client: %s", err)
			}
			var issuers googlecasissuerv1beta1.GoogleCASIssuerList
			err = client.List(ctx, &clients.GenericRequestOptions{}, &issuers)
			if err != nil {
				return nil, fmt.Errorf("failed to list cas issuers: %s", err)
			}
			for _, issuer := range issuers.Items {
				summaryIssuers = append(summaryIssuers, summaryIssuer{
					APIVersion: googlecasissuerv1beta1.GroupVersion.String(),
					Name:       issuer.Name,
					Namespace:  issuer.Namespace,
					Kind:       issuer.Kind,
				})
			}
		case clients.GoogleCASClusterIssuer:
			client, err := clients.NewGoogleCASClusterIssuerClient(cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to create cas cluster issuer client: %s", err)
			}
			var issuers googlecasissuerv1beta1.GoogleCASClusterIssuerList
			err = client.List(ctx, &clients.GenericRequestOptions{}, &issuers)
			if err != nil {
				return nil, fmt.Errorf("failed to list cas cluster issuers: %s", err)
			}
			for _, issuer := range issuers.Items {
				summaryIssuers = append(summaryIssuers, summaryIssuer{
					APIVersion: googlecasissuerv1beta1.GroupVersion.String(),
					Name:       issuer.Name,
					Kind:       issuer.Kind,
				})
			}
		case clients.AWSPCAIssuer:
			client, err := clients.NewAWSPCAIssuerClient(cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to create aws pca issuer client: %s", err)
			}
			var issuers awspcaissuerv1beta1.AWSPCAIssuerList
			err = client.List(ctx, &clients.GenericRequestOptions{}, &issuers)
			if err != nil {
				return nil, fmt.Errorf("failed to list pca issuers: %s", err)
			}
			for _, issuer := range issuers.Items {
				summaryIssuers = append(summaryIssuers, summaryIssuer{
					APIVersion: awspcaissuerv1beta1.GroupVersion.String(),
					Name:       issuer.Name,
					Namespace:  issuer.Namespace,
					Kind:       issuer.Kind,
				})
			}
		case clients.AWSPCAClusterIssuer:
			client, err := clients.NewAWSPCAClusterIssuerClient(cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to create aws pca cluster issuer client: %s", err)
			}
			var issuers awspcaissuerv1beta1.AWSPCAClusterIssuerList
			err = client.List(ctx, &clients.GenericRequestOptions{}, &issuers)
			if err != nil {
				return nil, fmt.Errorf("failed to list pca cluster issuers: %s", err)
			}
			for _, issuer := range issuers.Items {
				summaryIssuers = append(summaryIssuers, summaryIssuer{
					APIVersion: awspcaissuerv1beta1.GroupVersion.String(),
					Name:       issuer.Name,
					Kind:       issuer.Kind,
				})
			}
		case clients.KMSIssuer:
			client, err := clients.NewKMSIssuerClient(cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to create kms issuer client: %s", err)
			}
			var issuers kmsissuerv1alpha1.KMSIssuerList
			err = client.List(ctx, &clients.GenericRequestOptions{}, &issuers)
			if err != nil {
				return nil, fmt.Errorf("failed to list kms issuers: %s", err)
			}
			for _, issuer := range issuers.Items {
				summaryIssuers = append(summaryIssuers, summaryIssuer{
					APIVersion: kmsissuerv1alpha1.GroupVersion.String(),
					Name:       issuer.Name,
					Namespace:  issuer.Namespace,
					Kind:       issuer.Kind,
				})
			}
		case clients.VenafiEnhancedIssuer:
			client, err := clients.NewVenafiEnhancedIssuerClient(cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to create venafi enhanced issuer client: %s", err)
			}
			var issuers veiv1alpha1.VenafiIssuerList
			err = client.List(ctx, &clients.GenericRequestOptions{}, &issuers)
			if err != nil {
				return nil, fmt.Errorf("failed to list venafi issuers: %s", err)
			}
			for _, issuer := range issuers.Items {
				summaryIssuers = append(summaryIssuers, summaryIssuer{
					APIVersion: veiv1alpha1.SchemeGroupVersion.String(),
					Name:       issuer.Name,
					Namespace:  issuer.Namespace,
					Kind:       issuer.Kind,
				})
			}
		case clients.VenafiEnhancedClusterIssuer:
			client, err := clients.NewVenafiEnhancedClusterIssuerClient(cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to create venafi enhanced cluster issuer client: %s", err)
			}
			var issuers veiv1alpha1.VenafiClusterIssuerList
			err = client.List(ctx, &clients.GenericRequestOptions{}, &issuers)
			if err != nil {
				return nil, fmt.Errorf("failed to list venafi cluster issuers: %s", err)
			}
			for _, issuer := range issuers.Items {
				summaryIssuers = append(summaryIssuers, summaryIssuer{
					APIVersion: veiv1alpha1.SchemeGroupVersion.String(),
					Name:       issuer.Name,
					Kind:       issuer.Kind,
				})
			}
		case clients.OriginCAIssuer:
			client, err := clients.NewOriginCAIssuerClient(cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to create origin ca issuer client: %s", err)
			}
			var issuers origincaissuerv1.OriginIssuerList
			err = client.List(ctx, &clients.GenericRequestOptions{}, &issuers)
			if err != nil {
				return nil, fmt.Errorf("failed to list origin ca issuers: %s", err)
			}
			for _, issuer := range issuers.Items {
				summaryIssuers = append(summaryIssuers, summaryIssuer{
					APIVersion: kmsissuerv1alpha1.GroupVersion.String(),
					Name:       issuer.Name,
					Namespace:  issuer.Namespace,
					Kind:       issuer.Kind,
				})
			}
		case clients.SmallStepIssuer:
			client, err := clients.NewSmallStepIssuerClient(cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to create smallstep issuer client: %s", err)
			}
			var issuers stepissuerv1beta1.StepIssuerList
			err = client.List(ctx, &clients.GenericRequestOptions{}, &issuers)
			if err != nil {
				return nil, fmt.Errorf("failed to list smallstep issuers: %s", err)
			}
			for _, issuer := range issuers.Items {
				summaryIssuers = append(summaryIssuers, summaryIssuer{
					APIVersion: stepissuerv1beta1.GroupVersion.String(),
					Name:       issuer.Name,
					Namespace:  issuer.Namespace,
					Kind:       issuer.Kind,
				})
			}
		case clients.SmallStepClusterIssuer:
			client, err := clients.NewSmallStepClusterIssuerClient(cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to create smallstep cluster issuer client: %s", err)
			}
			var issuers stepissuerv1beta1.StepClusterIssuerList
			err = client.List(ctx, &clients.GenericRequestOptions{}, &issuers)
			if err != nil {
				return nil, fmt.Errorf("failed to list smallstep cluster issuers: %s", err)
			}
			for _, issuer := range issuers.Items {
				summaryIssuers = append(summaryIssuers, summaryIssuer{
					APIVersion: stepissuerv1beta1.GroupVersion.String(),
					Name:       issuer.Name,
					Namespace:  issuer.Namespace,
					Kind:       issuer.Kind,
				})
			}
		}
	}

	return summaryIssuers, nil
}

// findComponents takes a list of pods and returns a list of detected components
// which might be relevant to Jetstack Secure
func findComponents(md *components.MatchData) (map[string]installedComponent, error) {
	foundComponents := make(map[string]installedComponent)

	knownComponents := []installedComponent{
		&components.CertManagerStatus{},

		&components.CertManagerIstioCSRStatus{},

		&components.CertManagerCSIDriverStatus{},

		&components.CertManagerTrustManagerStatus{},

		&components.CertManagerCSIDriverSPIFFEStatus{},

		&components.CertManagerApproverPolicyStatus{},
		&components.CertManagerApproverPolicyEnterpriseStatus{},

		&components.JetstackSecureAgentStatus{},
		&components.JetstackSecureOperatorStatus{},

		&components.VenafiOAuthHelperStatus{},
		&components.CertDiscoveryVenafiStatus{},

		&components.VenafiEnhancedIssuerStatus{},
		&components.IsolatedIssuerStatus{},
		&components.GoogleCASIssuerStatus{},
		&components.AWSPCAIssuerStatus{},
		&components.KMSIssuerStatus{},
		&components.OriginCAIssuerStatus{},
		&components.SmallStepIssuerStatus{},
	}

	for i := range knownComponents {
		component := knownComponents[i]
		found, err := component.Match(md)
		if err != nil {
			return nil, fmt.Errorf("failed while testing pod as %s: %s", component.Name(), err)
		}
		if found {
			foundComponents[component.Name()] = component
		}
	}

	return foundComponents, nil
}
