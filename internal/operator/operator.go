// Package operator contains functions for installing and managing the jetstack operator.
package operator

import (
	"bytes"
	"context"
	"embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/Masterminds/semver"
	"github.com/cert-manager/cert-manager/pkg/apis/certmanager"
	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmanagermetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	operatorv1alpha1 "github.com/jetstack/js-operator/pkg/apis/operator/v1alpha1"
	"github.com/jetstack/jsctl/internal/docker"
	"github.com/jetstack/jsctl/internal/prompt"
	"github.com/jetstack/jsctl/internal/trustdomain"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/yaml"
)

// This embed.FS implementation contains every version of the installer YAML for the Jetstack Secure operator. Each
// one has been modified to act as a template so that fields such as the image registry can be modified. When adding
// a new version, place the full YAML file within the installers directory, and update the operator's Deployment
// resource to use a template field {{ .ImageRegistry }} as a suffix for the image name, rather than the default.
//go:embed installers/*.yaml
var installers embed.FS

// The Applier interface describes types that can Apply a stream of YAML-encoded Kubernetes resources.
type Applier interface {
	Apply(ctx context.Context, r io.Reader) error
}

// The ApplyOperatorYAMLOptions type contains fields used to configure the installation of the Jetstack Secure
// operator.
type ApplyOperatorYAMLOptions struct {
	Version             string // The version of the operator to use
	ImageRegistry       string // A custom image registry for the operator image
	CredentialsLocation string // The location of the service account key to access the Jetstack Secure image registry.
}

// ApplyOperatorYAML generates a YAML bundle that contains all Kubernetes resources required to run the Jetstack
// Secure operator which is then applied via the Applier implementation. It can be customised via the provided
// ApplyOperatorYAMLOptions type.
func ApplyOperatorYAML(ctx context.Context, applier Applier, options ApplyOperatorYAMLOptions) error {
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

	buf := bytes.NewBuffer([]byte{})
	if _, err = io.Copy(buf, file); err != nil {
		return err
	}

	secret, err := ImagePullSecret(options.CredentialsLocation)
	if err != nil {
		return err
	}

	buf.WriteString("---\n")

	if _, err = io.Copy(buf, secret); err != nil {
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

// ImagePullSecret returns an io.Reader implementation that contains the byte representation of the Kubernetes secret
// YAML that can be used as an image pull secret for the jetstack operator. The keyFileLocation parameter should describe
// the location of the authentication key file to use.
func ImagePullSecret(keyFileLocation string) (io.Reader, error) {
	file, err := os.Open(keyFileLocation)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return nil, ErrNoKeyFile
	case err != nil:
		return nil, err
	}
	defer file.Close()

	keyData, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	// When constructing a docker config for GCR, you must use the _json_key username and provide
	// any valid looking email address. Methodology for building this secret was taken from the kubectl
	// create secret command:
	// https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/kubectl/pkg/cmd/create/create_secret_docker.go
	const (
		username = "_json_key"
		email    = "auth@jetstack.io"
	)

	auth := username + ":" + string(keyData)
	config := docker.ConfigJSON{
		Auths: map[string]docker.ConfigEntry{
			"eu.gcr.io": {
				Username: username,
				Password: string(keyData),
				Email:    email,
				Auth:     base64.StdEncoding.EncodeToString([]byte(auth)),
			},
		},
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to encode docker config: %w", err)
	}

	const (
		secretName = "jse-gcr-creds"
		namespace  = "jetstack-secure"
	)

	secret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: configJSON,
		},
	}

	secretData, err := yaml.Marshal(secret)
	if err != nil {
		return nil, fmt.Errorf("failed to encode secret: %w", err)
	}

	return bytes.NewBuffer(secretData), nil
}

type (
	// The ApplyInstallationYAMLOptions type describes additional configuration options for the operator's Installation
	// custom resource.
	ApplyInstallationYAMLOptions struct {
		TrustDomains             map[string][]*trustdomain.TrustDomain // Specifies zero or more trust domains to use, keyed by namespace. A blank key assumes a ClusterIssuer.
		InstallCSIDriver         bool                                  // If true, the Installation manifest will have the cert-manager CSI driver.
		InstallSpiffeCSIDriver   bool                                  // If true, the Installation manifest will have the cert-manager spiffe CSI driver.
		InstallIstioCSR          bool                                  // If true, the Installation manifest will have the Istio CSR.
		InstallVenafiOauthHelper bool                                  // If true, the Installation manifest will have the venafi-oauth-helper.
		IstioCSRIssuer           string                                // The issuer name to use for the Istio CSR installation, should match the name of one of the TrustDomains.
		ImageRegistry            string                                // A custom image registry to use for operator components.
		Credentials              string                                // Path to a credentials file containing registry credentials for image pull secrets
		CertManagerReplicas      int                                   // The replica count for cert-manager and its components.
		IstioCSRReplicas         int                                   // The replica count for the istio-csr component.
		SpiffeCSIDriverReplicas  int                                   // The replica count for the csi-driver-spiffe component.

	}
)

// ApplyInstallationYAML generates a YAML bundle that describes the kubernetes manifest for the operator's Installation
// custom resource. The ApplyInstallationYAMLOptions specify additional options used to configure the installation.
func ApplyInstallationYAML(ctx context.Context, applier Applier, options ApplyInstallationYAMLOptions) error {
	apiVersion, kind := operatorv1alpha1.InstallationGVK.ToAPIVersionAndKind()

	installation := operatorv1alpha1.Installation{
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

	if err := applyTrustDomainsToInstallation(&installation, options.TrustDomains); err != nil {
		return fmt.Errorf("failed to configure trust domains: %w", err)
	}

	if err := applyIstioCSRToInstallation(&installation, options); err != nil {
		return fmt.Errorf("failed to configure istio csr: %w", err)
	}

	applyCSIDriversToInstallation(&installation, options)

	applyVenafiOauthHelperToInstallation(&installation, options)

	data, err := yaml.Marshal(installation)
	if err != nil {
		return fmt.Errorf("failed to encode installation: %w", err)
	}
	buf := bytes.NewBuffer(data)
	if options.Credentials != "" {
		secret, err := ImagePullSecret(options.Credentials)
		if err != nil {
			return fmt.Errorf("failed to parse image pull secret: %w", err)
		}
		buf.WriteString("---\n")
		if _, err = io.Copy(buf, secret); err != nil {
			return err
		}
	}

	return applier.Apply(ctx, buf)
}

func applyTrustDomainsToInstallation(installation *operatorv1alpha1.Installation, namespacedTrustDomains map[string][]*trustdomain.TrustDomain) error {
	for namespace, trustDomains := range namespacedTrustDomains {
		for _, trustDomain := range trustDomains {
			switch trustDomain.Type() {
			case trustdomain.TypeTPP:
				installation.Spec.Issuers = append(installation.Spec.Issuers, &operatorv1alpha1.Issuer{
					Name:         trustDomain.Name,
					ClusterScope: namespace == "",
					Namespace:    namespace,
					Venafi: &certmanagerv1.VenafiIssuer{
						Zone: trustDomain.TPP.Zone,
						TPP: &certmanagerv1.VenafiTPP{
							URL:      trustDomain.TPP.InstanceURL,
							CABundle: trustDomain.TPP.CABundle,
							CredentialsRef: certmanagermetav1.LocalObjectReference{
								Name: trustDomain.Name,
							},
						},
					},
				})
			default:
				return fmt.Errorf("unknown trust domain type: %s", trustDomain.Type())
			}
		}
	}

	return nil
}

func applyCSIDriversToInstallation(installation *operatorv1alpha1.Installation, options ApplyInstallationYAMLOptions) {
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
		installation.Spec.CSIDrivers = &drivers
	}
}

func applyIstioCSRToInstallation(installation *operatorv1alpha1.Installation, options ApplyInstallationYAMLOptions) error {
	if !options.InstallIstioCSR {
		return nil
	}

	installation.Spec.IstioCSR = &operatorv1alpha1.IstioCSR{
		ReplicaCount: &options.IstioCSRReplicas,
	}

	if options.IstioCSRIssuer == "" {
		return nil
	}

	// An installation can be configured to have multiple issuers via the trust domains. A single trust domain can be
	// chosen as the issuer to use for istio-csr.
	trustDomain, ok := findTrustDomain(options.TrustDomains, options.IstioCSRIssuer)
	if !ok {
		return fmt.Errorf("istio-csr issuer name does not match a provided trust domain")
	}

	installation.Spec.IstioCSR.IssuerRef = &certmanagermetav1.ObjectReference{
		Name:  trustDomain.Name,
		Kind:  certmanagerv1.IssuerKind,
		Group: certmanager.GroupName,
	}

	return nil
}

func applyVenafiOauthHelperToInstallation(installation *operatorv1alpha1.Installation, options ApplyInstallationYAMLOptions) error {
	if !options.InstallVenafiOauthHelper {
		return nil
	}

	var imagePullSecrets []string
	if options.Credentials != "" {
		imagePullSecrets = []string{"jse-gcr-creds"}
	}
	installation.Spec.VenafiOauthHelper = &operatorv1alpha1.VenafiOauthHelper{
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

func findTrustDomain(namespacedTrustDomains map[string][]*trustdomain.TrustDomain, name string) (*trustdomain.TrustDomain, bool) {
	for _, trustDomains := range namespacedTrustDomains {
		for _, trustDomain := range trustDomains {
			if trustDomain.Name == name {
				return trustDomain, true
			}
		}
	}

	return nil, false
}

// SuggestedActions generates a slice of prompt.Suggestion types based on the ApplyInstallationYAMLOptions. These are actions
// the user should perform to ensure that their installation works as expected.
func SuggestedActions(options ApplyInstallationYAMLOptions) []prompt.Suggestion {
	suggestions := make([]prompt.Suggestion, 0)

	for namespace, trustDomains := range options.TrustDomains {
		for _, trustDomain := range trustDomains {
			switch trustDomain.Type() {
			case trustdomain.TypeTPP:
				suggestions = append(suggestions,
					prompt.NewSuggestion(
						prompt.WithMessage("Trust domain '%s' requires a secret for configuration, please create one using your TPP access token", trustDomain.Name),
						prompt.WithCommand("jsctl trust-domain secret --for %s:%s $TPP_ACCESS_TOKEN", trustDomain.Name, namespace),
					))
			}
		}
	}

	if options.InstallIstioCSR {
		suggestions = append(suggestions,
			prompt.NewSuggestion(
				prompt.WithMessage("You can now install Istio and configure it to use istio-csr, follow the link below for examples"),
				prompt.WithLink("https://github.com/cert-manager/istio-csr/tree/main/hack"),
			))
	}

	return suggestions
}

type (
	// The InstallationClient is used to query information on an Installation resource within a Kubernetes cluster.
	InstallationClient struct {
		client *rest.RESTClient
	}

	// ComponentStatus describes the status of an individual operator component.
	ComponentStatus struct {
		Name    string `json:"name"`
		Ready   bool   `json:"ready"`
		Message string `json:"message,omitempty"`
	}
)

// NewInstallationClient returns a new instance of the InstallationClient that will interact with the Kubernetes
// cluster specified in the rest.Config.
func NewInstallationClient(config *rest.Config) (*InstallationClient, error) {
	// Set up the rest config to obtain Installation resources
	config.APIPath = "/apis"
	config.UserAgent = rest.DefaultKubernetesUserAgent()
	config.NegotiatedSerializer = serializer.NewCodecFactory(operatorv1alpha1.GlobalScheme)
	config.ContentConfig.GroupVersion = &schema.GroupVersion{
		Group:   operatorv1alpha1.InstallationGVK.Group,
		Version: operatorv1alpha1.InstallationGVK.Version,
	}

	restClient, err := rest.UnversionedRESTClientFor(config)
	if err != nil {
		return nil, err
	}

	return &InstallationClient{client: restClient}, nil
}

var (
	// ErrNoInstallation is the error given when querying an Installation resource that does not exist.
	ErrNoInstallation = errors.New("no installation")

	componentNames = map[operatorv1alpha1.InstallationConditionType]string{
		operatorv1alpha1.InstallationConditionCertManagerReady:        "cert-manager",
		operatorv1alpha1.InstallationConditionCertManagerIssuersReady: "issuers",
		operatorv1alpha1.InstallationConditionCSIDriversReady:         "csi-driver",
		operatorv1alpha1.InstallationConditionIstioCSRReady:           "istio-csr",
		operatorv1alpha1.InstallationConditionApproverPolicyReady:     "approver-policy",
		operatorv1alpha1.InstallationConditionVenafiOauthHelperReady:  "venafi-oauth-helper",
		operatorv1alpha1.InstallationConditionManifestsReady:          "manifests",
	}
)

// Status returns a slice of ComponentStatus types that describe the state of individual components installed by the
// operator. Returns ErrNoInstallation if an Installation resource cannot be found in the cluster. It uses the
// status conditions on an Installation resource and maps those to a ComponentStatus, the ComponentStatus.Name field
// is chosen based on the content of the componentNames map. Add friendly names to that map to include additional
// component statuses to return.
func (ic *InstallationClient) Status(ctx context.Context) ([]ComponentStatus, error) {
	var installation operatorv1alpha1.Installation

	const (
		resource = "installations"
		name     = "installation"
	)

	err := ic.client.Get().Resource(resource).Name(name).Do(ctx).Into(&installation)
	switch {
	case kerrors.IsNotFound(err):
		return nil, ErrNoInstallation
	case err != nil:
		return nil, err
	}

	statuses := make([]ComponentStatus, 0)
	for _, condition := range installation.Status.Conditions {
		componentStatus := ComponentStatus{
			Ready: condition.Status == operatorv1alpha1.ConditionTrue,
		}

		// Don't place the message if the component is considered ready.
		if !componentStatus.Ready {
			componentStatus.Message = condition.Message
		}

		// Swap the condition type for its friendly component name, don't include anything we don't have
		// a friendly name for.
		componentName, ok := componentNames[condition.Type]
		if !ok {
			continue
		}

		componentStatus.Name = componentName
		statuses = append(statuses, componentStatus)
	}

	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].Name < statuses[j].Name
	})

	return statuses, nil
}
