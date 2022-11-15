package clients

import (
	"fmt"

	v1alpha1approverpolicy "github.com/cert-manager/approver-policy/pkg/apis/policy/v1alpha1"
	v1certmanager "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	v1extensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/client-go/rest"
)

// NewCRDClient returns an instance of a generic client for querying CRDs
func NewCRDClient(config *rest.Config) (*Generic[*v1extensions.CustomResourceDefinition, *v1extensions.CustomResourceDefinitionList], error) {
	genericClient, err := NewGenericClient[*v1extensions.CustomResourceDefinition, *v1extensions.CustomResourceDefinitionList](
		&GenericClientOptions{
			RestConfig: config,
			APIPath:    "/apis",
			Group:      v1extensions.GroupName,
			Version:    v1extensions.SchemeGroupVersion.Version,
			Kind:       "customresourcedefinitions",
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error creating generic client: %w", err)
	}

	return genericClient, nil
}

// NewCertificateClient returns an instance of a generic client for querying cert-manager Certificates
func NewCertificateClient(config *rest.Config) (*Generic[*v1certmanager.Certificate, *v1certmanager.CertificateList], error) {
	genericClient, err := NewGenericClient[*v1certmanager.Certificate, *v1certmanager.CertificateList](
		&GenericClientOptions{
			RestConfig: config,
			APIPath:    "/apis",
			Group:      v1certmanager.SchemeGroupVersion.Group,
			Version:    v1certmanager.SchemeGroupVersion.Version,
			Kind:       "certificates",
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error creating generic client: %w", err)
	}

	return genericClient, nil
}

// NewCertificateRequestPolicyClient returns an instance of a generic client for
// querying approver policy CertificateRequestPolicies
func NewCertificateRequestPolicyClient(config *rest.Config) (*Generic[*v1alpha1approverpolicy.CertificateRequestPolicy, *v1alpha1approverpolicy.CertificateRequestPolicyList], error) {
	genericClient, err := NewGenericClient[*v1alpha1approverpolicy.CertificateRequestPolicy, *v1alpha1approverpolicy.CertificateRequestPolicyList](
		&GenericClientOptions{
			RestConfig: config,
			APIPath:    "/apis",
			Group:      v1alpha1approverpolicy.SchemeGroupVersion.Group,
			Version:    v1alpha1approverpolicy.SchemeGroupVersion.Version,
			Kind:       "certificaterequestpolicies",
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error creating generic client: %w", err)
	}

	return genericClient, nil
}
