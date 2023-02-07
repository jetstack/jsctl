package clients

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
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
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

type SupportedIssuer struct {
	CRDName  string
	Versions []string
}

type SupportedIssuerList []SupportedIssuer

func (s SupportedIssuerList) String() string {
	var items []string
	for _, issuer := range s {
		items = append(items, fmt.Sprintf("%s[%s]", issuer.CRDName, strings.Join(issuer.Versions, ",")))
	}
	return strings.Join(items, ", ")
}

// ListSupportedIssuers returns a static list of all supported issuer types and
// versions. It checks each items in the AllIssuers list is mapped and errors if
// there are missing Issuers. This is caught in a test to help us keep this up
// to date.
func ListSupportedIssuers() (SupportedIssuerList, error) {
	// this mapping must be manually updated if we change the supported issuers
	// and versions
	supportedGroupVersions := map[string][]string{
		CertManagerIssuer.String():           {"v1"},
		CertManagerClusterIssuer.String():    {"v1"},
		VenafiEnhancedIssuer.String():        {"v1alpha1"},
		VenafiEnhancedClusterIssuer.String(): {"v1alpha1"},
		AWSPCAIssuer.String():                {"v1beta1"},
		AWSPCAClusterIssuer.String():         {"v1beta1"},
		KMSIssuer.String():                   {"v1alpha1"},
		GoogleCASIssuer.String():             {"v1beta1"},
		GoogleCASClusterIssuer.String():      {"v1beta1"},
		OriginCAIssuer.String():              {"v1"},
		SmallStepIssuer.String():             {"v1beta1"},
		SmallStepClusterIssuer.String():      {"v1beta1"},
	}

	var supportedIssuers []SupportedIssuer

	// ensure that we have a list of versions for all supported issuers
	for _, issuer := range AllIssuersList {
		_, versionsKnown := supportedGroupVersions[issuer.String()]
		if !versionsKnown {
			return nil, fmt.Errorf("unknown issuer type %s", issuer.String())
		}
		supportedIssuers = append(supportedIssuers, SupportedIssuer{
			CRDName:  issuer.String(),
			Versions: supportedGroupVersions[issuer.String()],
		})
	}

	return supportedIssuers, nil
}

// AllIssuers is a special client to wrap logic for determining the kinds of
// issuers present in a cluster
type AllIssuers struct {
	crdClient Generic[*apiextensionsv1.CustomResourceDefinition, *apiextensionsv1.CustomResourceDefinitionList]
}

func (a *AllIssuers) ListKinds(ctx context.Context) ([]AnyIssuer, error) {
	// form an index of all known issuer types
	issuerIndex := make(map[string]AnyIssuer)
	for _, issuer := range AllIssuersList {
		issuerIndex[issuer.String()] = issuer
	}

	var crds apiextensionsv1.CustomResourceDefinitionList
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
func NewCertManagerIssuerClient(config *rest.Config) (Generic[*cmapi.Issuer, *cmapi.IssuerList], error) {
	genericClient, err := NewGenericClient[*cmapi.Issuer, *cmapi.IssuerList](
		&GenericClientOptions{
			RestConfig: config,
			APIPath:    "/apis",
			Group:      cmapi.SchemeGroupVersion.Group,
			Version:    cmapi.SchemeGroupVersion.Version,
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
func NewCertManagerClusterIssuerClient(config *rest.Config) (Generic[*cmapi.ClusterIssuer, *cmapi.ClusterIssuerList], error) {
	genericClient, err := NewGenericClient[*cmapi.ClusterIssuer, *cmapi.ClusterIssuerList](
		&GenericClientOptions{
			RestConfig: config,
			APIPath:    "/apis",
			Group:      cmapi.SchemeGroupVersion.Group,
			Version:    cmapi.SchemeGroupVersion.Version,
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
func NewGoogleCASIssuerClient(config *rest.Config) (Generic[*googlecasissuerv1beta1.GoogleCASIssuer, *googlecasissuerv1beta1.GoogleCASIssuerList], error) {
	genericClient, err := NewGenericClient[*googlecasissuerv1beta1.GoogleCASIssuer, *googlecasissuerv1beta1.GoogleCASIssuerList](
		&GenericClientOptions{
			RestConfig: config,
			APIPath:    "/apis",
			Group:      googlecasissuerv1beta1.GroupVersion.Group,
			Version:    googlecasissuerv1beta1.GroupVersion.Version,
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
func NewGoogleCASClusterIssuerClient(config *rest.Config) (Generic[*googlecasissuerv1beta1.GoogleCASClusterIssuer, *googlecasissuerv1beta1.GoogleCASClusterIssuerList], error) {
	genericClient, err := NewGenericClient[*googlecasissuerv1beta1.GoogleCASClusterIssuer, *googlecasissuerv1beta1.GoogleCASClusterIssuerList](
		&GenericClientOptions{
			RestConfig: config,
			APIPath:    "/apis",
			Group:      googlecasissuerv1beta1.GroupVersion.Group,
			Version:    googlecasissuerv1beta1.GroupVersion.Version,
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
func NewAWSPCAIssuerClient(config *rest.Config) (Generic[*awspcaissuerv1beta1.AWSPCAIssuer, *awspcaissuerv1beta1.AWSPCAIssuerList], error) {
	genericClient, err := NewGenericClient[*awspcaissuerv1beta1.AWSPCAIssuer, *awspcaissuerv1beta1.AWSPCAIssuerList](
		&GenericClientOptions{
			RestConfig: config,
			APIPath:    "/apis",
			Group:      awspcaissuerv1beta1.GroupVersion.Group,
			Version:    awspcaissuerv1beta1.GroupVersion.Version,
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
func NewAWSPCAClusterIssuerClient(config *rest.Config) (Generic[*awspcaissuerv1beta1.AWSPCAClusterIssuer, *awspcaissuerv1beta1.AWSPCAClusterIssuerList], error) {
	genericClient, err := NewGenericClient[*awspcaissuerv1beta1.AWSPCAClusterIssuer, *awspcaissuerv1beta1.AWSPCAClusterIssuerList](
		&GenericClientOptions{
			RestConfig: config,
			APIPath:    "/apis",
			Group:      awspcaissuerv1beta1.GroupVersion.Group,
			Version:    awspcaissuerv1beta1.GroupVersion.Version,
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
func NewKMSIssuerClient(config *rest.Config) (Generic[*kmsissuerv1alpha1.KMSIssuer, *kmsissuerv1alpha1.KMSIssuerList], error) {
	genericClient, err := NewGenericClient[*kmsissuerv1alpha1.KMSIssuer, *kmsissuerv1alpha1.KMSIssuerList](
		&GenericClientOptions{
			RestConfig: config,
			APIPath:    "/apis",
			Group:      kmsissuerv1alpha1.GroupVersion.Group,
			Version:    kmsissuerv1alpha1.GroupVersion.Version,
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
func NewVenafiEnhancedIssuerClient(config *rest.Config) (Generic[*veiv1alpha1.VenafiIssuer, *veiv1alpha1.VenafiIssuerList], error) {
	genericClient, err := NewGenericClient[*veiv1alpha1.VenafiIssuer, *veiv1alpha1.VenafiIssuerList](
		&GenericClientOptions{
			RestConfig: config,
			APIPath:    "/apis",
			Group:      veiv1alpha1.SchemeGroupVersion.Group,
			Version:    veiv1alpha1.SchemeGroupVersion.Version,
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
func NewVenafiEnhancedClusterIssuerClient(config *rest.Config) (Generic[*veiv1alpha1.VenafiClusterIssuer, *veiv1alpha1.VenafiClusterIssuerList], error) {
	genericClient, err := NewGenericClient[*veiv1alpha1.VenafiClusterIssuer, *veiv1alpha1.VenafiClusterIssuerList](
		&GenericClientOptions{
			RestConfig: config,
			APIPath:    "/apis",
			Group:      veiv1alpha1.SchemeGroupVersion.Group,
			Version:    veiv1alpha1.SchemeGroupVersion.Version,
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
func NewOriginCAIssuerClient(config *rest.Config) (Generic[*origincaissuerv1.OriginIssuer, *origincaissuerv1.OriginIssuerList], error) {
	genericClient, err := NewGenericClient[*origincaissuerv1.OriginIssuer, *origincaissuerv1.OriginIssuerList](
		&GenericClientOptions{
			RestConfig: config,
			APIPath:    "/apis",
			Group:      origincaissuerv1.GroupVersion.Group,
			Version:    origincaissuerv1.GroupVersion.Version,
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
func NewSmallStepIssuerClient(config *rest.Config) (Generic[*stepissuerv1beta1.StepIssuer, *stepissuerv1beta1.StepIssuerList], error) {
	genericClient, err := NewGenericClient[*stepissuerv1beta1.StepIssuer, *stepissuerv1beta1.StepIssuerList](
		&GenericClientOptions{
			RestConfig: config,
			APIPath:    "/apis",
			Group:      stepissuerv1beta1.GroupVersion.Group,
			Version:    stepissuerv1beta1.GroupVersion.Version,
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
func NewSmallStepClusterIssuerClient(config *rest.Config) (Generic[*stepissuerv1beta1.StepClusterIssuer, *stepissuerv1beta1.StepClusterIssuerList], error) {
	genericClient, err := NewGenericClient[*stepissuerv1beta1.StepClusterIssuer, *stepissuerv1beta1.StepClusterIssuerList](
		&GenericClientOptions{
			RestConfig: config,
			APIPath:    "/apis",
			Group:      stepissuerv1beta1.GroupVersion.Group,
			Version:    stepissuerv1beta1.GroupVersion.Version,
			Kind:       "stepclusterissuers",
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error creating generic client: %w", err)
	}
	return genericClient, nil
}
