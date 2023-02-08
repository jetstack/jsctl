package operator_test

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	certmanageracmev1 "github.com/cert-manager/cert-manager/pkg/apis/acme/v1"
	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmanagermetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	operatorv1alpha1 "github.com/jetstack/js-operator/pkg/apis/operator/v1alpha1"
	veiv1alpha1 "github.com/jetstack/venafi-enhanced-issuer/api/v1alpha1"
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

		assert.Error(t, operator.ApplyOperatorYAML(ctx, nil, opts))
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

	t.Run("It should interpolate image pull secret from file", func(t *testing.T) {
		applier := &TestApplier{}
		options := operator.ApplyInstallationYAMLOptions{
			RegistryCredentialsPath: "../registry/testdata/key.json",
		}

		err := operator.ApplyInstallationYAML(ctx, applier, options)
		assert.NoError(t, err)

		var secret corev1.Secret
		var actual operatorv1alpha1.Installation
		s := strings.Split(applier.data.String(), "---")
		assert.Len(t, s, 2)
		assert.NoError(t, yaml.Unmarshal([]byte(s[0]), &secret))
		assert.NoError(t, yaml.Unmarshal([]byte(s[1]), &actual))

		assert.Contains(t, actual.Spec.Images.Secret, "jse-gcr-creds")
	})
	t.Run("It should interpolate image pull secret from string contents", func(t *testing.T) {
		applier := &TestApplier{}
		options := operator.ApplyInstallationYAMLOptions{
			RegistryCredentials: "{'foo': 'bar'}",
		}

		err := operator.ApplyInstallationYAML(ctx, applier, options)
		assert.NoError(t, err)

		var secret corev1.Secret
		var actual operatorv1alpha1.Installation
		s := strings.Split(applier.data.String(), "---")
		assert.Len(t, s, 2)
		assert.NoError(t, yaml.Unmarshal([]byte(s[0]), &secret))
		assert.NoError(t, yaml.Unmarshal([]byte(s[1]), &actual))

		assert.Contains(t, actual.Spec.Images.Secret, "jse-gcr-creds")
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
		s := strings.Split(applier.data.String(), "---")
		assert.Len(t, s, 2)
		assert.NoError(t, yaml.Unmarshal([]byte(s[0]), &secret))
		assert.NoError(t, yaml.Unmarshal([]byte(s[1]), &installation))

		assert.NotNil(t, installation.Spec.CertDiscoveryVenafi)
		assert.Equal(t, installation.Spec.CertDiscoveryVenafi.TPP.URL, "foo")
		assert.Equal(t, installation.Spec.CertDiscoveryVenafi.TPP.Zone, "foozone")

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

			assert.EqualValues(t, cmapi.SchemeGroupVersion.Group, issuer.Group)
			assert.EqualValues(t, cmapi.IssuerKind, issuer.Kind)
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

		assert.EqualValues(t, options.ImageRegistry, actual.Spec.Images.Registry)
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
		certManagerIssuer := cmapi.Issuer{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cm-issuer-example",
				Namespace: "test-namespace",
			},
			Spec: cmapi.IssuerSpec{
				IssuerConfig: cmapi.IssuerConfig{
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

		certManagerClusterIssuer := cmapi.ClusterIssuer{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cm-cluster-issuer-example",
			},
			Spec: cmapi.IssuerSpec{
				IssuerConfig: cmapi.IssuerConfig{
					CA: &cmapi.CAIssuer{
						SecretName: "ca-key-pair",
					},
				},
			},
		}

		venafiIssuer := veiv1alpha1.VenafiIssuer{
			TypeMeta: metav1.TypeMeta{
				Kind:       "VenafiIssuer",
				APIVersion: "jetstack.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "application-team-b",
				Namespace: "test-namespace",
			},
			Spec: veiv1alpha1.VenafiCertificateSource{
				Tpp: &veiv1alpha1.TppCertificateIssuer{
					PolicyDn: `\VED\Policy\Teams\ApplicationTeamA`,
					Url:      "https://tpp1.example.com",
				},
			},
		}

		venafiClusterIssuer := veiv1alpha1.VenafiClusterIssuer{
			TypeMeta: metav1.TypeMeta{
				Kind:       "VenafiClusterIssuer",
				APIVersion: "jetstack.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "application-team-a",
			},
			Spec: veiv1alpha1.VenafiCertificateSource{
				Vaas: &veiv1alpha1.VaasCertificateIssuer{
					Application: "example",
					Template:    "example",
					ApiKey: []veiv1alpha1.SecretSource{
						{
							Secret: &veiv1alpha1.Secret{
								Name: "example",
							},
						},
					},
				},
			},
		}

		options := operator.ApplyInstallationYAMLOptions{
			ImportedCertManagerIssuers:        []*cmapi.Issuer{&certManagerIssuer},
			ImportedCertManagerClusterIssuers: []*cmapi.ClusterIssuer{&certManagerClusterIssuer},
			ImportedVenafiIssuers:             []*veiv1alpha1.VenafiIssuer{&venafiIssuer},
			ImportedVenafiClusterIssuers:      []*veiv1alpha1.VenafiClusterIssuer{&venafiClusterIssuer},
		}

		applier := &TestApplier{}
		err := operator.ApplyInstallationYAML(ctx, applier, options)
		assert.NoError(t, err)

		var actual operatorv1alpha1.Installation
		assert.NoError(t, yaml.Unmarshal(applier.data.Bytes(), &actual))

		expected := []*operatorv1alpha1.Issuer{
			{
				Name:      "cm-issuer-example",
				Namespace: "test-namespace",
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
			{
				Name:         "cm-cluster-issuer-example",
				ClusterScope: true,
				CA: &operatorv1alpha1.CAIssuer{
					SecretName: "ca-key-pair",
				},
			},
			{
				Name:      "application-team-b",
				Namespace: "test-namespace",
				VenafiEnhancedIssuer: &veiv1alpha1.VenafiCertificateSource{
					Tpp: &veiv1alpha1.TppCertificateIssuer{
						PolicyDn: `\VED\Policy\Teams\ApplicationTeamA`,
						Url:      "https://tpp1.example.com",
					},
				},
			},
			{
				Name:         "application-team-a",
				ClusterScope: true,
				VenafiEnhancedIssuer: &veiv1alpha1.VenafiCertificateSource{
					Vaas: &veiv1alpha1.VaasCertificateIssuer{
						Application: "example",
						Template:    "example",
						ApiKey: []veiv1alpha1.SecretSource{
							{
								Secret: &veiv1alpha1.Secret{
									Name: "example",
								},
							},
						},
					},
				},
			},
		}

		assert.NotNil(t, actual.Spec.Issuers)
		assert.Equal(t, expected, actual.Spec.Issuers)
	})
}
