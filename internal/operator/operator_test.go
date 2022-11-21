package operator_test

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	certmanageracmev1 "github.com/cert-manager/cert-manager/pkg/apis/acme/v1"
	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmanagermetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	operatorv1alpha1 "github.com/jetstack/js-operator/pkg/apis/operator/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/jetstack/jsctl/internal/operator"
	"github.com/jetstack/jsctl/internal/venafi"
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

func TestApplyInstallationYAML(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("It should generate an installation manifest", func(t *testing.T) {
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
	t.Run("It should add approver-policy-enterprise to the installation manifest", func(t *testing.T) {
		applier := &TestApplier{}
		options := operator.ApplyInstallationYAMLOptions{
			InstallApproverPolicyEnterprise: true,
		}

		err := operator.ApplyInstallationYAML(ctx, applier, options)
		assert.NoError(t, err)

		var actual operatorv1alpha1.Installation
		assert.NoError(t, yaml.Unmarshal(applier.data.Bytes(), &actual))

		assert.NotNil(t, actual.Spec.ApproverPolicyEnterprise)

		assert.Nil(t, actual.Spec.ApproverPolicy)
	})

	t.Run("It should add approver-policy-enterprise to the installation manifest and interpolate image pull secret", func(t *testing.T) {
		applier := &TestApplier{}
		options := operator.ApplyInstallationYAMLOptions{
			InstallApproverPolicyEnterprise: true,
			RegistryCredentialsPath:         "../registry/testdata/key.json",
		}

		err := operator.ApplyInstallationYAML(ctx, applier, options)
		assert.NoError(t, err)

		var secret corev1.Secret
		var actual operatorv1alpha1.Installation
		s := strings.Split(string(applier.data.Bytes()), "---")
		assert.Len(t, s, 2)
		assert.NoError(t, yaml.Unmarshal([]byte(s[0]), &secret))
		assert.NoError(t, yaml.Unmarshal([]byte(s[1]), &actual))

		assert.NotNil(t, actual.Spec.ApproverPolicyEnterprise)
		assert.Contains(t, actual.Spec.ApproverPolicyEnterprise.ImagePullSecrets, "jse-gcr-creds")
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
			RegistryCredentialsPath:  "../registry/testdata/key.json",
		}

		err := operator.ApplyInstallationYAML(ctx, applier, options)
		assert.NoError(t, err)

		var secret corev1.Secret
		var actual operatorv1alpha1.Installation
		s := strings.Split(string(applier.data.Bytes()), "---")
		assert.Len(t, s, 2)
		assert.NoError(t, yaml.Unmarshal([]byte(s[0]), &secret))
		assert.NoError(t, yaml.Unmarshal([]byte(s[1]), &actual))

		assert.NotNil(t, actual.Spec.VenafiOauthHelper)
		assert.Contains(t, actual.Spec.VenafiOauthHelper.ImagePullSecrets, "jse-gcr-creds")
	})

	t.Run("It should not add the cert-discovery-venafi to the installation manifest if it's not set ", func(t *testing.T) {
		applier := &TestApplier{}
		options := operator.ApplyInstallationYAMLOptions{}

		err := operator.ApplyInstallationYAML(ctx, applier, options)
		assert.NoError(t, err)

		var installation operatorv1alpha1.Installation
		assert.NoError(t, yaml.Unmarshal(applier.data.Bytes(), &installation))

		assert.Nil(t, installation.Spec.CertDiscoveryVenafi)

	})

	t.Run("It should add the cert-discovery-venafi to the installation manifest ", func(t *testing.T) {
		applier := &TestApplier{}
		cdv := &venafi.VenafiConnection{
			URL:         "foo",
			Zone:        "foozone",
			AccessToken: "footoken",
		}
		options := operator.ApplyInstallationYAMLOptions{
			CertDiscoveryVenafi: cdv,
		}

		err := operator.ApplyInstallationYAML(ctx, applier, options)
		assert.NoError(t, err)

		var secret corev1.Secret
		var installation operatorv1alpha1.Installation
		s := strings.Split(string(applier.data.Bytes()), "---")
		assert.Len(t, s, 2)
		assert.NoError(t, yaml.Unmarshal([]byte(s[0]), &secret))
		assert.NoError(t, yaml.Unmarshal([]byte(s[1]), &installation))

		assert.NotNil(t, installation.Spec.CertDiscoveryVenafi)
		assert.Equal(t, installation.Spec.CertDiscoveryVenafi.TPP.URL, "foo")
		assert.Equal(t, installation.Spec.CertDiscoveryVenafi.TPP.Zone, "foozone")

	})

	t.Run("It should add the cert-discovery-venafi to the installation manifest and interpolate image pull secret", func(t *testing.T) {
		applier := &TestApplier{}
		cdv := &venafi.VenafiConnection{
			URL:         "foo",
			Zone:        "foozone",
			AccessToken: "footoken",
		}
		options := operator.ApplyInstallationYAMLOptions{
			CertDiscoveryVenafi:     cdv,
			RegistryCredentialsPath: "../registry/testdata/key.json",
		}

		err := operator.ApplyInstallationYAML(ctx, applier, options)
		assert.NoError(t, err)
		var installation operatorv1alpha1.Installation
		s := strings.Split(string(applier.data.Bytes()), "---")
		assert.Len(t, s, 3)
		assert.NoError(t, yaml.Unmarshal([]byte(s[2]), &installation))

		assert.NotNil(t, installation.Spec.CertDiscoveryVenafi)
		assert.Contains(t, installation.Spec.CertDiscoveryVenafi.ImagePullSecrets, "jse-gcr-creds")
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

	t.Run("It should add the Istio CSR to the installation manifest", func(t *testing.T) {
		applier := &TestApplier{}
		options := operator.ApplyInstallationYAMLOptions{
			InstallIstioCSR: true,
			IstioCSRIssuer:  "example",
		}

		err := operator.ApplyInstallationYAML(ctx, applier, options)
		assert.NoError(t, err)

		var actual operatorv1alpha1.Installation
		assert.NoError(t, yaml.Unmarshal(applier.data.Bytes(), &actual))

		if assert.NotNil(t, actual.Spec.IstioCSR) && assert.NotNil(t, actual.Spec.IstioCSR.IssuerRef) {
			issuer := actual.Spec.IstioCSR.IssuerRef

			assert.EqualValues(t, certmanagerv1.SchemeGroupVersion.Group, issuer.Group)
			assert.EqualValues(t, certmanagerv1.IssuerKind, issuer.Kind)
			assert.EqualValues(t, options.IstioCSRIssuer, issuer.Name)
		}
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

	t.Run("It should generate an installation manifest with approver policy enterprise", func(t *testing.T) {
		options := operator.ApplyInstallationYAMLOptions{
			InstallApproverPolicyEnterprise: true,
		}
		applier := &TestApplier{}

		err := operator.ApplyInstallationYAML(ctx, applier, options)
		assert.NoError(t, err)

		var actual operatorv1alpha1.Installation
		assert.NoError(t, yaml.Unmarshal(applier.data.Bytes(), &actual))

		assert.Nil(t, actual.Spec.ApproverPolicy)
		assert.NotNil(t, actual.Spec.ApproverPolicyEnterprise)
	})

	t.Run("It should generate a manifest with issuers", func(t *testing.T) {
		iss := certmanagerv1.Issuer{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cm-issuer-example",
				Namespace: "test-namespace",
			},
			Spec: certmanagerv1.IssuerSpec{
				IssuerConfig: certmanagerv1.IssuerConfig{
					ACME: &certmanageracmev1.ACMEIssuer{
						Email:  "dummy-email@example.com",
						Server: "https://",
						PrivateKey: certmanagermetav1.SecretKeySelector{
							LocalObjectReference: certmanagermetav1.LocalObjectReference{
								Name: "example",
							},
						},
					},
				},
			},
		}

		clusterIss := certmanagerv1.ClusterIssuer{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cm-cluster-issuer-example",
				Namespace: "test-namespace",
			},
			Spec: certmanagerv1.IssuerSpec{
				IssuerConfig: certmanagerv1.IssuerConfig{
					CA: &certmanagerv1.CAIssuer{
						SecretName: "ca-key-pair",
					},
				},
			},
		}

		options := operator.ApplyInstallationYAMLOptions{
			CertManagerIssuers:        []*certmanagerv1.Issuer{&iss},
			CertManagerClusterIssuers: []*certmanagerv1.ClusterIssuer{&clusterIss},
		}

		applier := &TestApplier{}
		err := operator.ApplyInstallationYAML(ctx, applier, options)
		assert.NoError(t, err)

		var actual operatorv1alpha1.Installation
		assert.NoError(t, yaml.Unmarshal(applier.data.Bytes(), &actual))

		assert.NotNil(t, actual.Spec.Issuers)
		assert.Equal(t, 2, len(actual.Spec.Issuers))

		assert.Equal(t, "cm-issuer-example", actual.Spec.Issuers[0].Name)
		assert.Equal(t, "dummy-email@example.com", actual.Spec.Issuers[0].ACME.Email)
		assert.Equal(t, "cm-cluster-issuer-example", actual.Spec.Issuers[1].Name)
		assert.Equal(t, "ca-key-pair", actual.Spec.Issuers[1].CA.SecretName)
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
