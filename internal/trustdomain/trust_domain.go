// Package trustdomain contains functions for managing trust domains for an organization and their respective
// configurations.
package trustdomain

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"

	"github.com/jetstack/jsctl/internal/client"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

type (
	// The TrustDomain type describes a single trust domain and its configuration.
	TrustDomain struct {
		Name string            `json:"name"`
		TPP  *TPPConfiguration `json:"tpp,omitempty"`
	}

	// The HTTPClient interface describes types that perform HTTP requests.
	HTTPClient interface {
		Do(ctx context.Context, method, uri string, body, out interface{}) error
	}

	// The Type enumeration is used to distinguish between trust domain types.
	Type string
)

// Constants for trust domain types.
const (
	TypeUnknown = Type("UNKNOWN")
	TypeTPP     = Type("TPP")
)

// Type returns a string detailing which type of configuration a trust domain has.
func (td TrustDomain) Type() Type {
	switch {
	case td.TPP != nil:
		return TypeTPP
	default:
		return TypeUnknown
	}
}

// Create a new trust domain.
func Create(ctx context.Context, client HTTPClient, organization string, td TrustDomain) error {
	uri := path.Join("/api/v1/org", organization, "trust_domains")

	return client.Do(ctx, http.MethodPost, uri, td, nil)
}

// List all trust domains in an organization.
func List(ctx context.Context, client HTTPClient, organization string) ([]TrustDomain, error) {
	uri := path.Join("/api/v1/org", organization, "trust_domains")

	trustDomains := make([]TrustDomain, 0)
	if err := client.Do(ctx, http.MethodGet, uri, nil, &trustDomains); err != nil {
		return nil, err
	}

	return trustDomains, nil
}

// ErrNoTrustDomain is the error given when trying to delete a trust domain that does not exist in the organization.
var ErrNoTrustDomain = errors.New("no cluster")

// Delete a trust domain from the organization.
func Delete(ctx context.Context, httpClient HTTPClient, organization, name string) error {
	uri := path.Join("/api/v1/org", organization, "trust_domains", name)

	err := httpClient.Do(ctx, http.MethodDelete, uri, nil, nil)
	switch {
	case client.IsNotFound(err):
		return ErrNoTrustDomain
	case err != nil:
		return err
	default:
		return nil
	}
}

// Get a trust domain by name within the organization.
func Get(ctx context.Context, httpClient HTTPClient, organization, name string) (*TrustDomain, error) {
	uri := path.Join("/api/v1/org", organization, "trust_domains", name)

	var trustDomain TrustDomain
	err := httpClient.Do(ctx, http.MethodGet, uri, nil, &trustDomain)
	switch {
	case client.IsNotFound(err):
		return nil, ErrNoTrustDomain
	case err != nil:
		return nil, err
	default:
		return &trustDomain, err
	}
}

type (
	// ApplySecretOptions contains all configuration values used for different trust domain types to generate
	// a kubernetes secret.
	ApplySecretOptions struct {
		TPPAccessToken string // For TPP trust domains, the access token for the TPP instance.
		Namespace      string // The namespace the secret should belong to.
	}

	// The Applier interface describes types that can Apply a stream of YAML-encoded Kubernetes resources.
	Applier interface {
		Apply(ctx context.Context, r io.Reader) error
	}
)

// ErrUnknownTrustDomainType is the error returned when trying to perform an operation with a trust domain that does
// not have a supported type.
var ErrUnknownTrustDomainType = errors.New("unknown trust domain type")

// ApplySecret generates a Kubernetes Secret resource based on the provided trust domain and options. Once generated
// it is applied using the given Applier implementation.
func ApplySecret(ctx context.Context, applier Applier, domain *TrustDomain, opts ApplySecretOptions) error {
	const defaultNamespace = "cert-manager"

	secret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      domain.Name,
			Namespace: defaultNamespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: make(map[string][]byte),
	}

	if opts.Namespace != "" {
		secret.Namespace = opts.Namespace
	}

	switch domain.Type() {
	case TypeTPP:
		secret.Data["access-token"] = []byte(opts.TPPAccessToken)
	default:
		return fmt.Errorf("%w: %s", ErrUnknownTrustDomainType, domain.Type())
	}

	secretData, err := yaml.Marshal(secret)
	if err != nil {
		return fmt.Errorf("failed to encode secret: %w", err)
	}

	return applier.Apply(ctx, bytes.NewBuffer(secretData))
}
