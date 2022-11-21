// Package operator contains functions for installing and managing the jetstack operator.
package operator

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/Masterminds/semver"
	certmanageracmev1 "github.com/cert-manager/cert-manager/pkg/apis/acme/v1"
	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmanagermetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	operatorv1alpha1 "github.com/jetstack/js-operator/pkg/apis/operator/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/jetstack/jsctl/internal/prompt"
	"github.com/jetstack/jsctl/internal/registry"
	"github.com/jetstack/jsctl/internal/venafi"
)

// This embed.FS implementation contains every version of the installer YAML for the Jetstack Secure operator. Each
// one has been modified to act as a template so that fields such as the image registry can be modified. When adding
// a new version, place the full YAML file within the installers directory, and update the operator's Deployment
// resource to use a template field {{ .ImageRegistry }} as a suffix for the image name, rather than the default.
//
//go:embed installers/*.yaml
var installers embed.FS

// The Applier interface describes types that can Apply a stream of YAML-encoded Kubernetes resources.
type Applier interface {
	Apply(ctx context.Context, r io.Reader) error
}

// The ApplyOperatorYAMLOptions type contains fields used to configure the installation of the Jetstack Secure
// operator.
type ApplyOperatorYAMLOptions struct {
	Version       string // The version of the operator to use
	ImageRegistry string // A custom image registry for the operator image
	// RegistryCredentials is a string containing a GCP service account key to access the Jetstack Secure image registry.
	RegistryCredentials string
}

// ApplyOperatorYAML generates a YAML bundle that contains all Kubernetes resources required to run the Jetstack
// Secure operator which is then applied via the Applier implementation. It can be customised via the provided
// ApplyOperatorYAMLOptions type.
func ApplyOperatorYAML(ctx context.Context, applier Applier, options ApplyOperatorYAMLOptions) error {

	buf := bytes.NewBuffer([]byte{})

	// Write any secrets to the buffer first, so they get applied to cluster
	// before any Deployments that use them.
	// If there is no registry credentials, we assume that the images can be
	// pulled from a public registry or that the image pull secrets are already
	// in place.
	if options.RegistryCredentials != "" {
		secret, err := registry.ImagePullSecret(options.RegistryCredentials)
		if err != nil {
			return err
		}

		secretData, err := yaml.Marshal(secret)
		if err != nil {
			return fmt.Errorf("error marshalling secret data: %w", err)
		}
		secretReader := bytes.NewBuffer(secretData)

		if _, err = io.Copy(buf, secretReader); err != nil {
			return err
		}
		buf.WriteString("---\n")
	}

	var file io.Reader
	var err error

	if options.Version == "" {
		file, err = latestManifest()
	} else {
		file, err = manifestVersion(options.Version)
	}

	if err != nil {
		return err
	}

	if _, err = io.Copy(buf, file); err != nil {
		return err
	}

	tpl, err := template.New("install").Parse(buf.String())
	if err != nil {
		return err
	}

	output := bytes.NewBuffer([]byte{})
	err = tpl.Execute(output, map[string]interface{}{
		"ImageRegistry": options.ImageRegistry,
	})
	if err != nil {
		return err
	}

	return applier.Apply(ctx, output)
}

func latestManifest() (io.Reader, error) {
	versions, err := Versions()
	if err != nil {
		return nil, err
	}

	latest := versions[len(versions)-1]
	name := latest + ".yaml"

	return installers.Open(filepath.Join("installers", name))
}

// ErrNoManifest is the error given when querying a kubernetes manifest that doesn't exit.
var ErrNoManifest = errors.New("no manifest")

func manifestVersion(version string) (io.Reader, error) {
	name := version + ".yaml"
	file, err := installers.Open(filepath.Join("installers", name))
	switch {
	case errors.Is(err, os.ErrNotExist):
		return nil, ErrNoManifest
	case err != nil:
		return nil, err
	default:
		return file, nil
	}
}

// Versions returns all available versions of the jetstack operator ordered semantically.
func Versions() ([]string, error) {
	entries, err := installers.ReadDir("installers")
	if err != nil {
		return nil, err
	}

	rawVersions := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		rawVersion := strings.TrimSuffix(filepath.Base(entry.Name()), ".yaml")
		rawVersions = append(rawVersions, rawVersion)
	}

	parsedVersions := make([]*semver.Version, len(rawVersions))
	for i, rawVersion := range rawVersions {
		parsedVersion, err := semver.NewVersion(rawVersion)
		if err != nil {
			return nil, err
		}

		parsedVersions[i] = parsedVersion
	}

	sort.Sort(semver.Collection(parsedVersions))

	versions := make([]string, len(parsedVersions))
	for i, parsedVersion := range parsedVersions {
		versions[i] = "v" + parsedVersion.String()
	}

	return versions, nil
}

// ErrNoKeyFile is the error given when generating an image pull secret for a key that does not exist.
var ErrNoKeyFile = errors.New("no key file")

type (
	// The ApplyInstallationYAMLOptions type describes additional configuration options for the operator's Installation
	// custom resource.
	ApplyInstallationYAMLOptions struct {
		InstallCSIDriver       bool // If true, the Installation manifest will have the cert-manager CSI driver.
		InstallSpiffeCSIDriver bool // If true, the Installation manifest will have the cert-manager spiffe CSI driver.
		InstallIstioCSR        bool // If true, the Installation manifest will have the Istio CSR.
		// InstallApproverPolicyEnterprise, if true, will swap the default open
		// source policy approver for the enterprise one
		InstallApproverPolicyEnterprise bool
		CertDiscoveryVenafi             *venafi.VenafiConnection // If not nil, cert-discovery-venafi resources will be added to manifests
		InstallVenafiOauthHelper        bool                     // If true, the Installation manifest will have the venafi-oauth-helper.
		VenafiIssuers                   []*venafi.VenafiIssuer
		IstioCSRIssuer                  string // The issuer name to use for the Istio CSR installation.
		ImageRegistry                   string // A custom image registry to use for operator components.
		RegistryCredentialsPath         string // Path to a credentials file containing registry credentials for image pull secrets
		// RegistryCredentials is a string containing a GCP service account key to access the Jetstack Secure image registry.
		RegistryCredentials     string
		CertManagerReplicas     int    // The replica count for cert-manager and its components.
		CertManagerVersion      string // The version of cert-manager to deploy
		IstioCSRReplicas        int    // The replica count for the istio-csr component.
		SpiffeCSIDriverReplicas int    // The replica count for the csi-driver-spiffe component.

		// CertManagerIssuers is a list of cert-manager issuers to include in
		// the generated installation file
		CertManagerIssuers []*certmanagerv1.Issuer
		// CertManagerClusterIssuers is a list of cert-manager cluster issuers
		// to include in the generated installation file
		CertManagerClusterIssuers []*certmanagerv1.ClusterIssuer
	}
)

// ApplyInstallationYAML generates a YAML bundle that describes the kubernetes manifest for the operator's Installation
// custom resource. The ApplyInstallationYAMLOptions specify additional options used to configure the installation.
func ApplyInstallationYAML(ctx context.Context, applier Applier, options ApplyInstallationYAMLOptions) error {
	apiVersion, kind := operatorv1alpha1.InstallationGVK.ToAPIVersionAndKind()

	installation := &operatorv1alpha1.Installation{
		TypeMeta: metav1.TypeMeta{
			Kind:       kind,
			APIVersion: apiVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "installation",
		},
		Spec: operatorv1alpha1.InstallationSpec{
			Registry: options.ImageRegistry,
			CertManager: &operatorv1alpha1.CertManager{
				Controller: &operatorv1alpha1.CertManagerControllerConfig{
					ReplicaCount: &options.CertManagerReplicas,
				},
				Webhook: &operatorv1alpha1.CertManagerWebhookConfig{
					ReplicaCount: &options.CertManagerReplicas,
				},
			},
			ApproverPolicy: &operatorv1alpha1.ApproverPolicy{},
		},
	}
	manifestTemplates := &manifests{
		installation: installation,
		secrets:      make([]*corev1.Secret, 0),
	}

	if err := applyIstioCSRToInstallation(manifestTemplates, options); err != nil {
		return fmt.Errorf("failed to configure istio csr: %w", err)
	}

	applyApproverPolicyEnterpriseToInstallation(manifestTemplates, options)

	applyCertManagerVersion(manifestTemplates, options)

	applyCSIDriversToInstallation(manifestTemplates, options)

	applyVenafiOauthHelperToInstallation(manifestTemplates, options)

	applyCertDiscoveryVenafiManifests(manifestTemplates, options)

	registryCredentials := options.RegistryCredentials
	if registryCredentials == "" && options.RegistryCredentialsPath != "" {
		registryCredentialsBytes, err := os.ReadFile(options.RegistryCredentialsPath)
		if err != nil {
			return fmt.Errorf("failed to read registry credentials file: %w", err)
		}
		registryCredentials = string(registryCredentialsBytes)
	}

	if registryCredentials != "" {
		secret, err := registry.ImagePullSecret(registryCredentials)
		if err != nil {
			return fmt.Errorf("failed to parse image pull secret: %w", err)
		}
		manifestTemplates.secrets = append(manifestTemplates.secrets, secret)
	}

	if err := generateVenafiIssuerManifests(manifestTemplates, options); err != nil {
		return fmt.Errorf("error building manifests for Venafi issuers: %w", err)
	}

	err := addIssuersToInstallation(
		manifestTemplates,
		options.CertManagerIssuers,
		options.CertManagerClusterIssuers,
	)
	if err != nil {
		return fmt.Errorf("error adding issuers to installation: %w", err)
	}

	buf, err := marshalManifests(manifestTemplates)
	if err != nil {
		return fmt.Errorf("error marshalling manifests: %w", err)
	}

	return applier.Apply(ctx, buf)
}

func addIssuersToInstallation(mf *manifests, issuers []*certmanagerv1.Issuer, clusterIssuers []*certmanagerv1.ClusterIssuer) error {
	for _, issuer := range issuers {
		mf.installation.Spec.Issuers = append(mf.installation.Spec.Issuers, &operatorv1alpha1.Issuer{
			ClusterScope: false,
			Name:         issuer.Name,
			Namespace:    issuer.Namespace,
			Labels:       issuer.Labels,
			Annotations:  issuer.Annotations,

			ACME: func() *certmanageracmev1.ACMEIssuer {
				if issuer.Spec.ACME != nil {
					return issuer.Spec.ACME
				}
				return nil
			}(),

			CA: func() *operatorv1alpha1.CAIssuer {
				if issuer.Spec.CA != nil {
					return &operatorv1alpha1.CAIssuer{
						SecretName:            issuer.Spec.CA.SecretName,
						CRLDistributionPoints: issuer.Spec.CA.CRLDistributionPoints,
						OCSPServers:           issuer.Spec.CA.OCSPServers,
					}
				}
				return nil
			}(),

			Vault: func() *certmanagerv1.VaultIssuer {
				if issuer.Spec.Vault != nil {
					return issuer.Spec.Vault
				}
				return nil
			}(),
			SelfSigned: func() *certmanagerv1.SelfSignedIssuer {
				if issuer.Spec.SelfSigned != nil {
					return issuer.Spec.SelfSigned
				}
				return nil
			}(),
			Venafi: func() *certmanagerv1.VenafiIssuer {
				if issuer.Spec.Venafi != nil {
					return issuer.Spec.Venafi
				}
				return nil
			}(),
		})
	}

	for _, issuer := range clusterIssuers {
		mf.installation.Spec.Issuers = append(mf.installation.Spec.Issuers, &operatorv1alpha1.Issuer{
			ClusterScope: true,
			Name:         issuer.Name,
			Namespace:    issuer.Namespace,
			Labels:       issuer.Labels,
			Annotations:  issuer.Annotations,

			ACME: func() *certmanageracmev1.ACMEIssuer {
				if issuer.Spec.ACME != nil {
					return issuer.Spec.ACME
				}
				return nil
			}(),

			CA: func() *operatorv1alpha1.CAIssuer {
				if issuer.Spec.CA != nil {
					return &operatorv1alpha1.CAIssuer{
						SecretName:            issuer.Spec.CA.SecretName,
						CRLDistributionPoints: issuer.Spec.CA.CRLDistributionPoints,
						OCSPServers:           issuer.Spec.CA.OCSPServers,
					}
				}
				return nil
			}(),

			Vault: func() *certmanagerv1.VaultIssuer {
				if issuer.Spec.Vault != nil {
					return issuer.Spec.Vault
				}
				return nil
			}(),
			SelfSigned: func() *certmanagerv1.SelfSignedIssuer {
				if issuer.Spec.SelfSigned != nil {
					return issuer.Spec.SelfSigned
				}
				return nil
			}(),
			Venafi: func() *certmanagerv1.VenafiIssuer {
				if issuer.Spec.Venafi != nil {
					return issuer.Spec.Venafi
				}
				return nil
			}(),
		})
	}

	return nil
}

func generateVenafiIssuerManifests(mf *manifests, options ApplyInstallationYAMLOptions) error {
	for _, issuerTemplate := range options.VenafiIssuers {
		issuer, secret, err := venafi.GenerateOperatorManifestsForIssuer(issuerTemplate)
		if err != nil {
			return fmt.Errorf("error generating manifests for Venafi issuer: %w", err)
		}
		mf.secrets = append(mf.secrets, secret)
		mf.installation.Spec.Issuers = append(mf.installation.Spec.Issuers, issuer)
	}

	return nil
}

func applyCertDiscoveryVenafiManifests(mf *manifests, options ApplyInstallationYAMLOptions) {
	if options.CertDiscoveryVenafi == nil {
		return
	}
	cdv, secret := venafi.GenerateManifestsForCertDiscoveryVenafi(options.CertDiscoveryVenafi)
	var imagePullSecrets []string
	if options.RegistryCredentialsPath != "" || options.RegistryCredentials != "" {
		imagePullSecrets = []string{"jse-gcr-creds"}
	}
	// Eventually we probably want to have a single field for image pull
	// secrets on Installation resource, but this change will happen in the
	// operator.
	if cdv != nil {
		cdv.ImagePullSecrets = imagePullSecrets
	}
	mf.installation.Spec.CertDiscoveryVenafi = cdv
	mf.secrets = append(mf.secrets, secret)
}

type manifests struct {
	installation *operatorv1alpha1.Installation
	secrets      []*corev1.Secret
}

func marshalManifests(mf *manifests) (io.Reader, error) {
	buf := bytes.NewBuffer([]byte{})

	// Add all Secrets to the buffer first to ensure that they get applied
	// to the cluster before any Deployments that might want to use them.
	for _, secret := range mf.secrets {
		secretYAML, err := yaml.Marshal(secret)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal Secret data: %w", err)
		}
		secretReader := bytes.NewBuffer(secretYAML)
		if _, err = io.Copy(buf, secretReader); err != nil {
			return nil, fmt.Errorf("error writing secret data to buffer: %w", err)
		}
		buf.WriteString("---\n")
	}
	if mf.installation.Spec.CertManager == nil {
		panic("cert manager is nil")
	}
	installationData, err := yaml.Marshal(mf.installation)
	if err != nil {
		return nil, fmt.Errorf("error marshalling Installation resource: %w", err)
	}

	installationBuffer := bytes.NewReader(installationData)

	if _, err = io.Copy(buf, installationBuffer); err != nil {
		return nil, fmt.Errorf("Error writing installation data to buffer: %w", err)
	}

	return buf, nil
}

func applyVenafiIssuerResources(manifestTemplates *manifests, options ApplyOperatorYAMLOptions) error {
	return nil
}

func applyCertManagerVersion(manifestTemplates *manifests, options ApplyInstallationYAMLOptions) {
	if options.CertManagerVersion == "" {
		return
	}
	manifestTemplates.installation.Spec.CertManager.Version = options.CertManagerVersion
}

func applyCSIDriversToInstallation(manifests *manifests, options ApplyInstallationYAMLOptions) {
	var assign bool
	var drivers operatorv1alpha1.CSIDrivers

	// The validating webhook will reject installation.Spec.CSIDrivers being non-null if there is not at least one
	// CSI driver enabled. So we check each option and set a boolean to know if we should instantiate it.
	if options.InstallCSIDriver {
		assign = true
		drivers.CertManager = &operatorv1alpha1.CSIDriverCertManager{}
	}

	if options.InstallSpiffeCSIDriver {
		assign = true
		drivers.CertManagerSpiffe = &operatorv1alpha1.CSIDriverCertManagerSpiffe{
			ReplicaCount: &options.SpiffeCSIDriverReplicas,
		}
	}

	if assign {
		manifests.installation.Spec.CSIDrivers = &drivers
	}
}

func applyIstioCSRToInstallation(manifests *manifests, options ApplyInstallationYAMLOptions) error {
	if !options.InstallIstioCSR {
		return nil
	}

	manifests.installation.Spec.IstioCSR = &operatorv1alpha1.IstioCSR{
		ReplicaCount: &options.IstioCSRReplicas,
	}

	if options.IstioCSRIssuer == "" {
		return nil
	}

	manifests.installation.Spec.IstioCSR.IssuerRef = &certmanagermetav1.ObjectReference{
		Name:  options.IstioCSRIssuer,
		Kind:  certmanagerv1.IssuerKind,
		Group: certmanagerv1.SchemeGroupVersion.Group,
	}

	return nil
}

func applyVenafiOauthHelperToInstallation(manifests *manifests, options ApplyInstallationYAMLOptions) error {
	if !options.InstallVenafiOauthHelper {
		return nil
	}

	var imagePullSecrets []string
	if options.RegistryCredentials != "" || options.RegistryCredentialsPath != "" {
		imagePullSecrets = []string{"jse-gcr-creds"}
	}
	manifests.installation.Spec.VenafiOauthHelper = &operatorv1alpha1.VenafiOauthHelper{
		ImagePullSecrets: imagePullSecrets,
	}

	return nil
}
func applyApproverPolicyEnterpriseToInstallation(manifests *manifests, options ApplyInstallationYAMLOptions) error {
	if !options.InstallApproverPolicyEnterprise {
		return nil
	}

	manifests.installation.Spec.ApproverPolicy = nil

	var imagePullSecrets []string
	if options.RegistryCredentials != "" || options.RegistryCredentialsPath != "" {
		imagePullSecrets = []string{"jse-gcr-creds"}
	}
	manifests.installation.Spec.ApproverPolicyEnterprise = &operatorv1alpha1.ApproverPolicyEnterprise{
		ImagePullSecrets: imagePullSecrets,
	}

	return nil
}

func applyImagePullSecrets(installation *operatorv1alpha1.Installation, options ApplyInstallationYAMLOptions) error {
	if !options.InstallVenafiOauthHelper {
		return nil
	}

	installation.Spec.VenafiOauthHelper = &operatorv1alpha1.VenafiOauthHelper{}

	return nil
}

// SuggestedActions generates a slice of prompt.Suggestion types based on the ApplyInstallationYAMLOptions. These are actions
// the user should perform to ensure that their installation works as expected.
func SuggestedActions(options ApplyInstallationYAMLOptions) []prompt.Suggestion {
	suggestions := make([]prompt.Suggestion, 0)

	if options.InstallIstioCSR {
		suggestions = append(suggestions,
			prompt.NewSuggestion(
				prompt.WithMessage("You can now install Istio and configure it to use istio-csr, follow the link below for examples"),
				prompt.WithLink("https://github.com/cert-manager/istio-csr/tree/main/hack"),
			))
	}

	return suggestions
}
