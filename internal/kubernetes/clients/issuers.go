package clients

import (
	"context"
	"fmt"

	v1alpha1kmsissuer "github.com/Skyscanner/kms-issuer/apis/certmanager/v1alpha1"
	v1beta1awspcaissuer "github.com/cert-manager/aws-privateca-issuer/pkg/api/v1beta1"
	v1certmanager "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	v1origincaissuer "github.com/cloudflare/origin-ca-issuer/pkgs/apis/v1"
	v1beta1googlecasissuer "github.com/jetstack/google-cas-issuer/api/v1beta1"
	v1alpha1vei "github.com/jetstack/venafi-enhanced-issuer/api/v1alpha1"
	v1beta1stepissuer "github.com/smallstep/step-issuer/api/v1beta1"
	v1apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/client-go/rest"
)

// AnyIssuer is an enum of all known issuer types, external and built-in.
type AnyIssuer int64

const (
	CertManagerIssuer AnyIssuer = iota
	CertManagerClusterIssuer
	VenafiEnhancedIssuer
	VenafiEnhancedClusterIssuer
	AWSPCAIssuer
	AWSPCAClusterIssuer
	KMSIssuer
	GoogleCASIssuer
	GoogleCASClusterIssuer
	OriginCAIssuer
	SmallStepIssuer
	SmallStepClusterIssuer
)

var AllIssuersList = []AnyIssuer{
	CertManagerIssuer,
	CertManagerClusterIssuer,
	VenafiEnhancedIssuer,
	VenafiEnhancedClusterIssuer,
	AWSPCAIssuer,
	AWSPCAClusterIssuer,
	KMSIssuer,
	GoogleCASIssuer,
	GoogleCASClusterIssuer,
	OriginCAIssuer,
	SmallStepIssuer,
	SmallStepClusterIssuer,
}

func (s AnyIssuer) String() string {
	switch s {
	case CertManagerIssuer:
		return "issuers.cert-manager.io"
	case CertManagerClusterIssuer:
		return "clusterissuers.cert-manager.io"
	case VenafiEnhancedIssuer:
		return "venafiissuers.jetstack.io"
	case VenafiEnhancedClusterIssuer:
		return "venaficlusterissuers.jetstack.io"
	case AWSPCAIssuer:
		return "awspcaissuers.awspca.cert-manager.io"
	case AWSPCAClusterIssuer:
		return "awspcaclusterissuers.awspca.cert-manager.io"
	case KMSIssuer:
		return "kmsissuers.cert-manager.skyscanner.net"
	case GoogleCASIssuer:
		return "googlecasissuers.cas-issuer.jetstack.io"
	case GoogleCASClusterIssuer:
		return "googlecasclusterissuers.cas-issuer.jetstack.io"
	case OriginCAIssuer:
		return "originissuers.cert-manager.k8s.cloudflare.com"
	case SmallStepIssuer:
		return "stepissuers.certmanager.step.sm"
	case SmallStepClusterIssuer:
		return "stepclusterissuers.certmanager.step.sm"
	}
	return "unknown"
}

// AllIssuers is a special client to wrap logic for determining the kinds of
// issuers present in a cluster
type AllIssuers struct {
	crdClient *Generic[*v1apiextensions.CustomResourceDefinition, *v1apiextensions.CustomResourceDefinitionList]
}

func (a *AllIssuers) ListKinds(ctx context.Context) ([]AnyIssuer, error) {
	// form an index of all known issuer types
	issuerIndex := make(map[string]AnyIssuer)
	for _, issuer := range AllIssuersList {
		issuerIndex[issuer.String()] = issuer
	}

	var crds v1apiextensions.CustomResourceDefinitionList
	err := a.crdClient.List(ctx, &GenericRequestOptions{}, &crds)

	if err != nil {
		return nil, fmt.Errorf("error listing CRDs: %w", err)
	}

	var foundIssuers []AnyIssuer
	for _, crd := range crds.Items {
		anyIssuer, ok := issuerIndex[crd.Name]
		if ok {
			foundIssuers = append(foundIssuers, anyIssuer)
		}
	}

	return foundIssuers, nil
}

// NewAllIssuers returns a new instance of and AllIssuers client.
func NewAllIssuers(config *rest.Config) (*AllIssuers, error) {
	crdClient, err := NewCRDClient(config)
	if err != nil {
		return nil, fmt.Errorf("error creating CRD client: %w", err)
	}

	return &AllIssuers{
		crdClient: crdClient,
	}, nil
}

// NewCertManagerIssuerClient returns an instance of a generic client for querying
// cert-manager Issuers
func NewCertManagerIssuerClient(config *rest.Config) (*Generic[*v1certmanager.Issuer, *v1certmanager.IssuerList], error) {
	genericClient, err := NewGenericClient[*v1certmanager.Issuer, *v1certmanager.IssuerList](
		&GenericClientOptions{
			RestConfig: config,
			APIPath:    "/apis",
			Group:      v1certmanager.SchemeGroupVersion.Group,
			Version:    v1certmanager.SchemeGroupVersion.Version,
			Kind:       "issuers",
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error creating generic client: %w", err)
	}

	return genericClient, nil
}

// NewCertManagerClusterIssuerClient returns an instance of a generic client
// for querying cert-manager ClusterIssuers
func NewCertManagerClusterIssuerClient(config *rest.Config) (*Generic[*v1certmanager.ClusterIssuer, *v1certmanager.ClusterIssuerList], error) {
	genericClient, err := NewGenericClient[*v1certmanager.ClusterIssuer, *v1certmanager.ClusterIssuerList](
		&GenericClientOptions{
			RestConfig: config,
			APIPath:    "/apis",
			Group:      v1certmanager.SchemeGroupVersion.Group,
			Version:    v1certmanager.SchemeGroupVersion.Version,
			Kind:       "clusterissuers",
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error creating generic client: %w", err)
	}

	return genericClient, nil
}

// NewGoogleCASIssuerClient returns an instance of a generic client for querying
// google CAS Issuers
func NewGoogleCASIssuerClient(config *rest.Config) (*Generic[*v1beta1googlecasissuer.GoogleCASIssuer, *v1beta1googlecasissuer.GoogleCASIssuerList], error) {
	genericClient, err := NewGenericClient[*v1beta1googlecasissuer.GoogleCASIssuer, *v1beta1googlecasissuer.GoogleCASIssuerList](
		&GenericClientOptions{
			RestConfig: config,
			APIPath:    "/apis",
			Group:      v1beta1googlecasissuer.GroupVersion.Group,
			Version:    v1beta1googlecasissuer.GroupVersion.Version,
			Kind:       "googlecasissuers",
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error creating generic client: %w", err)
	}

	return genericClient, nil
}

// NewGoogleCASClusterIssuerClient returns an instance of a generic client for querying
// google CAS cluster Issuers
func NewGoogleCASClusterIssuerClient(config *rest.Config) (*Generic[*v1beta1googlecasissuer.GoogleCASClusterIssuer, *v1beta1googlecasissuer.GoogleCASClusterIssuerList], error) {
	genericClient, err := NewGenericClient[*v1beta1googlecasissuer.GoogleCASClusterIssuer, *v1beta1googlecasissuer.GoogleCASClusterIssuerList](
		&GenericClientOptions{
			RestConfig: config,
			APIPath:    "/apis",
			Group:      v1beta1googlecasissuer.GroupVersion.Group,
			Version:    v1beta1googlecasissuer.GroupVersion.Version,
			Kind:       "googlecasclusterissuers",
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error creating generic client: %w", err)
	}

	return genericClient, nil
}

// NewAWSPCAIssuerClient returns an instance of a generic client for querying
// AWS PCA Issuers
func NewAWSPCAIssuerClient(config *rest.Config) (*Generic[*v1beta1awspcaissuer.AWSPCAIssuer, *v1beta1awspcaissuer.AWSPCAIssuerList], error) {
	genericClient, err := NewGenericClient[*v1beta1awspcaissuer.AWSPCAIssuer, *v1beta1awspcaissuer.AWSPCAIssuerList](
		&GenericClientOptions{
			RestConfig: config,
			APIPath:    "/apis",
			Group:      v1beta1awspcaissuer.GroupVersion.Group,
			Version:    v1beta1awspcaissuer.GroupVersion.Version,
			Kind:       "awspcaissuers",
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error creating generic client: %w", err)
	}

	return genericClient, nil
}

// NewAWSPCAClusterIssuerClient returns an instance of a generic client for querying
// AWS PCA cluster Issuers
func NewAWSPCAClusterIssuerClient(config *rest.Config) (*Generic[*v1beta1awspcaissuer.AWSPCAClusterIssuer, *v1beta1awspcaissuer.AWSPCAClusterIssuerList], error) {
	genericClient, err := NewGenericClient[*v1beta1awspcaissuer.AWSPCAClusterIssuer, *v1beta1awspcaissuer.AWSPCAClusterIssuerList](
		&GenericClientOptions{
			RestConfig: config,
			APIPath:    "/apis",
			Group:      v1beta1awspcaissuer.GroupVersion.Group,
			Version:    v1beta1awspcaissuer.GroupVersion.Version,
			Kind:       "awspcaclusterissuers",
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error creating generic client: %w", err)
	}

	return genericClient, nil
}

// NewKMSIssuerClient returns an instance of a generic client for querying
// KMS Issuers
func NewKMSIssuerClient(config *rest.Config) (*Generic[*v1alpha1kmsissuer.KMSIssuer, *v1alpha1kmsissuer.KMSIssuerList], error) {
	genericClient, err := NewGenericClient[*v1alpha1kmsissuer.KMSIssuer, *v1alpha1kmsissuer.KMSIssuerList](
		&GenericClientOptions{
			RestConfig: config,
			APIPath:    "/apis",
			Group:      v1alpha1kmsissuer.GroupVersion.Group,
			Version:    v1alpha1kmsissuer.GroupVersion.Version,
			Kind:       "kmsissuers",
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error creating generic client: %w", err)
	}

	return genericClient, nil
}

// NewVenafiEnhancedIssuerClient returns an instance of a generic client for querying
// Venafi enhanced issuers
func NewVenafiEnhancedIssuerClient(config *rest.Config) (*Generic[*v1alpha1vei.VenafiIssuer, *v1alpha1vei.VenafiIssuerList], error) {
	genericClient, err := NewGenericClient[*v1alpha1vei.VenafiIssuer, *v1alpha1vei.VenafiIssuerList](
		&GenericClientOptions{
			RestConfig: config,
			APIPath:    "/apis",
			Group:      v1alpha1vei.SchemeGroupVersion.Group,
			Version:    v1alpha1vei.SchemeGroupVersion.Group,
			Kind:       "venafiissuers",
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error creating generic client: %w", err)
	}

	return genericClient, nil
}

// NewVenafiEnhancedClusterIssuerClient returns an instance of a generic client for querying
// Venafi enhanced cluster issuers
func NewVenafiEnhancedClusterIssuerClient(config *rest.Config) (*Generic[*v1alpha1vei.VenafiClusterIssuer, *v1alpha1vei.VenafiClusterIssuerList], error) {
	genericClient, err := NewGenericClient[*v1alpha1vei.VenafiClusterIssuer, *v1alpha1vei.VenafiClusterIssuerList](
		&GenericClientOptions{
			RestConfig: config,
			APIPath:    "/apis",
			Group:      v1alpha1vei.SchemeGroupVersion.Group,
			Version:    v1alpha1vei.SchemeGroupVersion.Group,
			Kind:       "venaficlusterissuers",
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error creating generic client: %w", err)
	}

	return genericClient, nil
}

// NewOriginCAIssuerClient returns an instance of a generic client for querying
// Origin CA Issuers
func NewOriginCAIssuerClient(config *rest.Config) (*Generic[*v1origincaissuer.OriginIssuer, *v1origincaissuer.OriginIssuerList], error) {
	genericClient, err := NewGenericClient[*v1origincaissuer.OriginIssuer, *v1origincaissuer.OriginIssuerList](
		&GenericClientOptions{
			RestConfig: config,
			APIPath:    "/apis",
			Group:      v1origincaissuer.GroupVersion.Group,
			Version:    v1origincaissuer.GroupVersion.Version,
			Kind:       "originissuers",
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error creating generic client: %w", err)
	}

	return genericClient, nil
}

// NewSmallStepIssuerClient returns an instance of a generic client for querying
// Step Issuers
func NewSmallStepIssuerClient(config *rest.Config) (*Generic[*v1beta1stepissuer.StepIssuer, *v1beta1stepissuer.StepIssuerList], error) {
	genericClient, err := NewGenericClient[*v1beta1stepissuer.StepIssuer, *v1beta1stepissuer.StepIssuerList](
		&GenericClientOptions{
			RestConfig: config,
			APIPath:    "/apis",
			Group:      v1beta1stepissuer.GroupVersion.Group,
			Version:    v1beta1stepissuer.GroupVersion.Version,
			Kind:       "stepissuers",
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error creating generic client: %w", err)
	}

	return genericClient, nil
}

// NewSmallStepClusterIssuerClient returns an instance of a generic client for querying
// Step Cluster Issuers
func NewSmallStepClusterIssuerClient(config *rest.Config) (*Generic[*v1beta1stepissuer.StepClusterIssuer, *v1beta1stepissuer.StepClusterIssuerList], error) {
	genericClient, err := NewGenericClient[*v1beta1stepissuer.StepClusterIssuer, *v1beta1stepissuer.StepClusterIssuerList](
		&GenericClientOptions{
			RestConfig: config,
			APIPath:    "/apis",
			Group:      v1beta1stepissuer.GroupVersion.Group,
			Version:    v1beta1stepissuer.GroupVersion.Version,
			Kind:       "stepclusterissuers",
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error creating generic client: %w", err)
	}
	return genericClient, nil
}
