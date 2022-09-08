package operator_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/cert-manager/cert-manager/pkg/apis/certmanager"
	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	operatorv1alpha1 "github.com/jetstack/js-operator/pkg/apis/operator/v1alpha1"
	"github.com/jetstack/jsctl/internal/docker"
	"github.com/jetstack/jsctl/internal/operator"
	"github.com/jetstack/jsctl/internal/trustdomain"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

type (
	TestApplier struct {
		data *bytes.Buffer
	}
)

func (ta *TestApplier) Apply(_ context.Context, r io.Reader) error {
	if ta.data == nil {
		ta.data = bytes.NewBuffer([]byte{})
	}

	_, err := io.Copy(ta.data, r)
	return err
}

func TestApplyOperatorYAML(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("It should return an error for a version that does not exist", func(t *testing.T) {
		opts := operator.ApplyOperatorYAMLOptions{
			Version: "v99.99.99",
		}

		err := operator.ApplyOperatorYAML(ctx, nil, opts)
		assert.Equal(t, operator.ErrNoManifest, err)
	})
}

func TestVersions(t *testing.T) {
	t.Parallel()

	versions, err := operator.Versions()
	assert.NoError(t, err)
	assert.NotEmpty(t, versions)
}

func TestImagePullSecret(t *testing.T) {
	t.Parallel()

	t.Run("It should load valid credentials and generate a secret", func(t *testing.T) {
		data, err := operator.ImagePullSecret("./testdata/key.json")
		assert.NoError(t, err)

		var actual corev1.Secret
		assert.NoError(t, yaml.Unmarshal(data.(*bytes.Buffer).Bytes(), &actual))

		assert.EqualValues(t, "jetstack-secure", actual.Namespace)
		assert.EqualValues(t, "jse-gcr-creds", actual.Name)
		assert.EqualValues(t, corev1.SecretTypeDockerConfigJson, actual.Type)
		assert.NotEmpty(t, actual.Data[corev1.DockerConfigJsonKey])

		var actualConfig docker.ConfigJSON
		assert.NoError(t, json.Unmarshal(actual.Data[corev1.DockerConfigJsonKey], &actualConfig))
		assert.NotEmpty(t, actualConfig.Auths)

		actualGCR := actualConfig.Auths["eu.gcr.io"]
		assert.NotEmpty(t, actualGCR.Email)
		assert.NotEmpty(t, actualGCR.Password)
		assert.NotEmpty(t, actualGCR.Auth)
		assert.NotEmpty(t, actualGCR.Username)
	})

	t.Run("It should return an error if the key file does not exist", func(t *testing.T) {
		data, err := operator.ImagePullSecret("./testdata/nope.json")
		assert.Equal(t, operator.ErrNoKeyFile, err)
		assert.Nil(t, data)
	})
}

func TestApplyInstallationYAML(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("It should generate an installation manifest with no trust domain", func(t *testing.T) {
		options := operator.ApplyInstallationYAMLOptions{}
		applier := &TestApplier{}

		err := operator.ApplyInstallationYAML(ctx, applier, options)
		assert.NoError(t, err)

		var actual operatorv1alpha1.Installation
		assert.NoError(t, yaml.Unmarshal(applier.data.Bytes(), &actual))

		assert.NotEmpty(t, actual.Name)
		assert.NotEmpty(t, actual.Kind)
		assert.NotEmpty(t, actual.APIVersion)
		assert.NotNil(t, actual.Spec.CertManager)
		assert.NotNil(t, actual.Spec.ApproverPolicy)
		assert.Nil(t, actual.Spec.CSIDrivers)
		assert.Len(t, actual.Spec.Issuers, 0)
	})

	t.Run("It should generate an installation manifest with a cluster-scoped TPP trust domain", func(t *testing.T) {
		applier := &TestApplier{}
		options := operator.ApplyInstallationYAMLOptions{
			TrustDomains: map[string][]*trustdomain.TrustDomain{
				"": {
					{
						Name: "example",
						TPP: &trustdomain.TPPConfiguration{
							Zone:        "example",
							InstanceURL: "https://example.com",
						},
					},
				},
			},
		}

		err := operator.ApplyInstallationYAML(ctx, applier, options)
		assert.NoError(t, err)

		var actual operatorv1alpha1.Installation
		assert.NoError(t, yaml.Unmarshal(applier.data.Bytes(), &actual))

		assert.NotEmpty(t, actual.Name)
		assert.NotEmpty(t, actual.Kind)
		assert.NotEmpty(t, actual.APIVersion)
		assert.NotNil(t, actual.Spec.CertManager)
		assert.NotNil(t, actual.Spec.ApproverPolicy)
		assert.Nil(t, actual.Spec.CSIDrivers)

		if assert.Len(t, actual.Spec.Issuers, 1) {
			issuer := actual.Spec.Issuers[0]

			assert.NotEmpty(t, issuer.Name)
			assert.Empty(t, issuer.Namespace)
			assert.NotNil(t, issuer.Venafi)
			assert.NotNil(t, issuer.Venafi.TPP)
			assert.True(t, issuer.ClusterScope)
			assert.Equal(t, options.TrustDomains[""][0].TPP.Zone, issuer.Venafi.Zone)
			assert.Equal(t, options.TrustDomains[""][0].TPP.InstanceURL, issuer.Venafi.TPP.URL)
			assert.Equal(t, options.TrustDomains[""][0].Name, issuer.Venafi.TPP.CredentialsRef.Name)
		}
	})

	t.Run("It should generate an installation manifest with a namespace-scoped TPP trust domain", func(t *testing.T) {
		applier := &TestApplier{}
		options := operator.ApplyInstallationYAMLOptions{
			TrustDomains: map[string][]*trustdomain.TrustDomain{
				"my-namespace": {
					{
						Name: "example",
						TPP: &trustdomain.TPPConfiguration{
							Zone:        "example",
							InstanceURL: "https://example.com",
						},
					},
				},
			},
		}

		err := operator.ApplyInstallationYAML(ctx, applier, options)
		assert.NoError(t, err)

		var actual operatorv1alpha1.Installation
		assert.NoError(t, yaml.Unmarshal(applier.data.Bytes(), &actual))

		assert.NotEmpty(t, actual.Name)
		assert.NotEmpty(t, actual.Kind)
		assert.NotEmpty(t, actual.APIVersion)
		assert.NotNil(t, actual.Spec.CertManager)
		assert.NotNil(t, actual.Spec.ApproverPolicy)
		assert.Nil(t, actual.Spec.CSIDrivers)

		if assert.Len(t, actual.Spec.Issuers, 1) {
			issuer := actual.Spec.Issuers[0]

			assert.NotEmpty(t, issuer.Name)
			assert.NotNil(t, issuer.Venafi)
			assert.NotNil(t, issuer.Venafi.TPP)
			assert.False(t, issuer.ClusterScope)
			assert.Equal(t, "my-namespace", issuer.Namespace)
			assert.Equal(t, options.TrustDomains["my-namespace"][0].TPP.Zone, issuer.Venafi.Zone)
			assert.Equal(t, options.TrustDomains["my-namespace"][0].TPP.InstanceURL, issuer.Venafi.TPP.URL)
			assert.Equal(t, options.TrustDomains["my-namespace"][0].Name, issuer.Venafi.TPP.CredentialsRef.Name)
		}
	})

	t.Run("It should generate an installation manifest with multiple cluster-scoped TPP trust domains", func(t *testing.T) {
		applier := &TestApplier{}
		options := operator.ApplyInstallationYAMLOptions{
			TrustDomains: map[string][]*trustdomain.TrustDomain{
				"": {
					{
						Name: "first",
						TPP: &trustdomain.TPPConfiguration{
							Zone:        "example",
							InstanceURL: "https://example.com",
						},
					},
					{
						Name: "second",
						TPP: &trustdomain.TPPConfiguration{
							Zone:        "example",
							InstanceURL: "https://example.com",
						},
					},
				},
			},
		}

		err := operator.ApplyInstallationYAML(ctx, applier, options)
		assert.NoError(t, err)

		var actual operatorv1alpha1.Installation
		assert.NoError(t, yaml.Unmarshal(applier.data.Bytes(), &actual))

		assert.NotEmpty(t, actual.Name)
		assert.NotEmpty(t, actual.Kind)
		assert.NotEmpty(t, actual.APIVersion)
		assert.NotNil(t, actual.Spec.CertManager)
		assert.NotNil(t, actual.Spec.ApproverPolicy)
		assert.Nil(t, actual.Spec.CSIDrivers)

		if assert.Len(t, actual.Spec.Issuers, 2) {
			first := findIssuer(t, "first", "", actual.Spec.Issuers)
			second := findIssuer(t, "second", "", actual.Spec.Issuers)

			assert.Equal(t, options.TrustDomains[""][0].Name, first.Name)
			assert.Empty(t, first.Namespace)
			assert.NotNil(t, first.Venafi)
			assert.NotNil(t, first.Venafi.TPP)
			assert.True(t, first.ClusterScope)
			assert.Equal(t, options.TrustDomains[""][0].TPP.Zone, first.Venafi.Zone)
			assert.Equal(t, options.TrustDomains[""][0].TPP.InstanceURL, first.Venafi.TPP.URL)
			assert.Equal(t, options.TrustDomains[""][0].Name, first.Venafi.TPP.CredentialsRef.Name)

			assert.Equal(t, options.TrustDomains[""][1].Name, second.Name)
			assert.Empty(t, second.Namespace)
			assert.NotNil(t, second.Venafi)
			assert.NotNil(t, second.Venafi.TPP)
			assert.True(t, second.ClusterScope)
			assert.Equal(t, options.TrustDomains[""][1].TPP.Zone, second.Venafi.Zone)
			assert.Equal(t, options.TrustDomains[""][1].TPP.InstanceURL, second.Venafi.TPP.URL)
			assert.Equal(t, options.TrustDomains[""][1].Name, second.Venafi.TPP.CredentialsRef.Name)
		}
	})

	t.Run("It should generate an installation manifest with both namespace and cluster scoped TPP trust domains", func(t *testing.T) {
		applier := &TestApplier{}
		options := operator.ApplyInstallationYAMLOptions{
			TrustDomains: map[string][]*trustdomain.TrustDomain{
				"": {
					{
						Name: "first",
						TPP: &trustdomain.TPPConfiguration{
							Zone:        "example",
							InstanceURL: "https://example.com",
						},
					},
				},
				"my-namespace": {
					{
						Name: "second",
						TPP: &trustdomain.TPPConfiguration{
							Zone:        "example",
							InstanceURL: "https://example.com",
						},
					},
				},
			},
		}

		err := operator.ApplyInstallationYAML(ctx, applier, options)
		assert.NoError(t, err)

		var actual operatorv1alpha1.Installation
		assert.NoError(t, yaml.Unmarshal(applier.data.Bytes(), &actual))

		assert.NotEmpty(t, actual.Name)
		assert.NotEmpty(t, actual.Kind)
		assert.NotEmpty(t, actual.APIVersion)
		assert.NotNil(t, actual.Spec.CertManager)
		assert.NotNil(t, actual.Spec.ApproverPolicy)
		assert.Nil(t, actual.Spec.CSIDrivers)

		if assert.Len(t, actual.Spec.Issuers, 2) {
			first := findIssuer(t, "first", "", actual.Spec.Issuers)
			second := findIssuer(t, "second", "my-namespace", actual.Spec.Issuers)

			assert.Equal(t, options.TrustDomains[""][0].Name, first.Name)
			assert.Empty(t, first.Namespace)
			assert.NotNil(t, first.Venafi)
			assert.NotNil(t, first.Venafi.TPP)
			assert.True(t, first.ClusterScope)
			assert.Equal(t, options.TrustDomains[""][0].TPP.Zone, first.Venafi.Zone)
			assert.Equal(t, options.TrustDomains[""][0].TPP.InstanceURL, first.Venafi.TPP.URL)
			assert.Equal(t, options.TrustDomains[""][0].Name, first.Venafi.TPP.CredentialsRef.Name)

			assert.Equal(t, options.TrustDomains["my-namespace"][0].Name, second.Name)
			assert.NotNil(t, second.Venafi)
			assert.NotNil(t, second.Venafi.TPP)
			assert.False(t, second.ClusterScope)
			assert.Equal(t, options.TrustDomains["my-namespace"][0].TPP.Zone, second.Venafi.Zone)
			assert.Equal(t, options.TrustDomains["my-namespace"][0].TPP.InstanceURL, second.Venafi.TPP.URL)
			assert.Equal(t, options.TrustDomains["my-namespace"][0].Name, second.Venafi.TPP.CredentialsRef.Name)
		}
	})

	t.Run("It should add the CSI driver to the installation manifest", func(t *testing.T) {
		applier := &TestApplier{}
		options := operator.ApplyInstallationYAMLOptions{
			InstallCSIDriver: true,
		}

		err := operator.ApplyInstallationYAML(ctx, applier, options)
		assert.NoError(t, err)

		var actual operatorv1alpha1.Installation
		assert.NoError(t, yaml.Unmarshal(applier.data.Bytes(), &actual))

		if assert.NotNil(t, actual.Spec.CSIDrivers) {
			assert.NotNil(t, actual.Spec.CSIDrivers.CertManager)
		}
	})

	t.Run("It should add the venafi-oauth-helper to the installation manifest", func(t *testing.T) {
		applier := &TestApplier{}
		options := operator.ApplyInstallationYAMLOptions{
			InstallVenafiOauthHelper: true,
		}

		err := operator.ApplyInstallationYAML(ctx, applier, options)
		assert.NoError(t, err)

		var actual operatorv1alpha1.Installation
		assert.NoError(t, yaml.Unmarshal(applier.data.Bytes(), &actual))

		assert.NotNil(t, actual.Spec.VenafiOauthHelper)
	})

	t.Run("It should add the venafi-oauth-helper to the installation manifest and interpolate image pull secret", func(t *testing.T) {
		applier := &TestApplier{}
		options := operator.ApplyInstallationYAMLOptions{
			InstallVenafiOauthHelper: true,
			Credentials:              "./testdata/key.json",
		}

		err := operator.ApplyInstallationYAML(ctx, applier, options)
		assert.NoError(t, err)

		var actual operatorv1alpha1.Installation
		assert.NoError(t, yaml.Unmarshal(applier.data.Bytes(), &actual))

		assert.NotNil(t, actual.Spec.VenafiOauthHelper)
		assert.Contains(t, actual.Spec.VenafiOauthHelper.ImagePullSecrets, "jse-gcr-creds")
	})

	t.Run("It should have a blank Istio CSR block when no issuer is provided", func(t *testing.T) {
		applier := &TestApplier{}
		options := operator.ApplyInstallationYAMLOptions{
			InstallIstioCSR: true,
		}

		err := operator.ApplyInstallationYAML(ctx, applier, options)
		assert.NoError(t, err)

		var actual operatorv1alpha1.Installation
		assert.NoError(t, yaml.Unmarshal(applier.data.Bytes(), &actual))

		assert.NotNil(t, actual.Spec.IstioCSR)
		assert.NotNil(t, actual.Spec.IstioCSR.ReplicaCount)
	})

	t.Run("It should add the Istio CSR to the installation manifest with a namespaced trust domain as the issuer", func(t *testing.T) {
		applier := &TestApplier{}
		options := operator.ApplyInstallationYAMLOptions{
			InstallIstioCSR: true,
			IstioCSRIssuer:  "example",
			TrustDomains: map[string][]*trustdomain.TrustDomain{
				"": {
					{
						Name: "example",
						TPP: &trustdomain.TPPConfiguration{
							Zone:        "example",
							InstanceURL: "https://example.com",
						},
					},
				},
			},
		}

		err := operator.ApplyInstallationYAML(ctx, applier, options)
		assert.NoError(t, err)

		var actual operatorv1alpha1.Installation
		assert.NoError(t, yaml.Unmarshal(applier.data.Bytes(), &actual))

		if assert.NotNil(t, actual.Spec.IstioCSR) && assert.NotNil(t, actual.Spec.IstioCSR.IssuerRef) {
			issuer := actual.Spec.IstioCSR.IssuerRef

			assert.EqualValues(t, certmanager.GroupName, issuer.Group)
			assert.EqualValues(t, certmanagerv1.IssuerKind, issuer.Kind)
			assert.EqualValues(t, options.IstioCSRIssuer, issuer.Name)
		}
	})

	t.Run("It should return an error when setting the Istio CSR issuer to a non-existant trust domain", func(t *testing.T) {
		options := operator.ApplyInstallationYAMLOptions{
			InstallIstioCSR: true,
			IstioCSRIssuer:  "doesnt-exist",
			TrustDomains: map[string][]*trustdomain.TrustDomain{
				"": {
					{
						Name: "example",
						TPP: &trustdomain.TPPConfiguration{
							Zone:        "example",
							InstanceURL: "https://example.com",
						},
					},
				},
			},
		}

		err := operator.ApplyInstallationYAML(ctx, nil, options)
		assert.Error(t, err)
	})

	t.Run("It should set the image registry for components", func(t *testing.T) {
		applier := &TestApplier{}
		options := operator.ApplyInstallationYAMLOptions{
			ImageRegistry: "ghcr.io/my-org",
		}

		err := operator.ApplyInstallationYAML(ctx, applier, options)
		assert.NoError(t, err)

		var actual operatorv1alpha1.Installation
		assert.NoError(t, yaml.Unmarshal(applier.data.Bytes(), &actual))

		assert.EqualValues(t, options.ImageRegistry, actual.Spec.Registry)
	})

	t.Run("It should include the spiffe CSI driver", func(t *testing.T) {
		applier := &TestApplier{}
		options := operator.ApplyInstallationYAMLOptions{
			InstallSpiffeCSIDriver: true,
		}

		err := operator.ApplyInstallationYAML(ctx, applier, options)
		assert.NoError(t, err)

		var actual operatorv1alpha1.Installation
		assert.NoError(t, yaml.Unmarshal(applier.data.Bytes(), &actual))

		assert.NotNil(t, actual.Spec.CSIDrivers.CertManagerSpiffe)
	})

	t.Run("It should specify the replica count for cert-manager", func(t *testing.T) {
		applier := &TestApplier{}
		options := operator.ApplyInstallationYAMLOptions{
			CertManagerReplicas: 2,
		}

		err := operator.ApplyInstallationYAML(ctx, applier, options)
		assert.NoError(t, err)

		var actual operatorv1alpha1.Installation
		assert.NoError(t, yaml.Unmarshal(applier.data.Bytes(), &actual))

		if assert.NotNil(t, actual.Spec.CertManager.Webhook) {
			assert.EqualValues(t, &options.CertManagerReplicas, actual.Spec.CertManager.Webhook.ReplicaCount)
		}

		if assert.NotNil(t, actual.Spec.CertManager.Controller) {
			assert.EqualValues(t, &options.CertManagerReplicas, actual.Spec.CertManager.Controller.ReplicaCount)
		}
	})

	t.Run("It should specify the replica count for csi-driver-spiffe", func(t *testing.T) {
		applier := &TestApplier{}
		options := operator.ApplyInstallationYAMLOptions{
			InstallSpiffeCSIDriver:  true,
			SpiffeCSIDriverReplicas: 2,
		}

		err := operator.ApplyInstallationYAML(ctx, applier, options)
		assert.NoError(t, err)

		var actual operatorv1alpha1.Installation
		assert.NoError(t, yaml.Unmarshal(applier.data.Bytes(), &actual))

		if assert.NotNil(t, actual.Spec.CSIDrivers.CertManagerSpiffe) {
			assert.EqualValues(t, &options.SpiffeCSIDriverReplicas, actual.Spec.CSIDrivers.CertManagerSpiffe.ReplicaCount)
		}
	})

	t.Run("It should specify the replica count for istio-csr", func(t *testing.T) {
		applier := &TestApplier{}
		options := operator.ApplyInstallationYAMLOptions{
			InstallIstioCSR:  true,
			IstioCSRReplicas: 2,
		}

		err := operator.ApplyInstallationYAML(ctx, applier, options)
		assert.NoError(t, err)

		var actual operatorv1alpha1.Installation
		assert.NoError(t, yaml.Unmarshal(applier.data.Bytes(), &actual))

		if assert.NotNil(t, actual.Spec.IstioCSR) {
			assert.EqualValues(t, &options.IstioCSRReplicas, actual.Spec.IstioCSR.ReplicaCount)
		}
	})
}

func findIssuer(t *testing.T, name, namespace string, issuers []*operatorv1alpha1.Issuer) *operatorv1alpha1.Issuer {
	t.Helper()

	for _, issuer := range issuers {
		if issuer.Name == name && issuer.Namespace == namespace {
			return issuer
		}
	}

	assert.Fail(t, "invalid issuer lookup")
	return nil
}
