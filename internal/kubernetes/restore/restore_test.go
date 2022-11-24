package restore

import (
	"testing"

	certmanageracmev1 "github.com/cert-manager/cert-manager/pkg/apis/acme/v1"
	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmanagermetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	veiv1alpha1 "github.com/jetstack/venafi-enhanced-issuer/api/v1alpha1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestExtractOperatorManageableIssuersFromBackupFile(t *testing.T) {
	testCases := map[string]struct {
		backupFilePath string
	}{
		"yaml": {
			backupFilePath: "fixtures/backup.yaml",
		},
		"json": {
			backupFilePath: "fixtures/backup.json",
		},
	}

	expectedIssuers := &RestoredIssuers{
		MissedIssuers: []string{
			"AWSPCAIssuer/pca-sample",
			"GoogleCASIssuer/googlecasissuer-sample",
		},
		CertManagerIssuers: []*certmanagerv1.Issuer{
			{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Issuer",
					APIVersion: "cert-manager.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cm-issuer-sample",
					Namespace: "jetstack-secure",
				},
				Spec: certmanagerv1.IssuerSpec{
					IssuerConfig: certmanagerv1.IssuerConfig{
						CA: &certmanagerv1.CAIssuer{
							SecretName: "ca-key-pair",
						},
					},
				},
			},
		},
		CertManagerClusterIssuers: []*certmanagerv1.ClusterIssuer{
			{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ClusterIssuer",
					APIVersion: "cert-manager.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "cm-cluster-issuer-sample",
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
			},
		},
		VenafiIssuers: []*veiv1alpha1.VenafiIssuer{
			{
				TypeMeta: metav1.TypeMeta{
					Kind:       "VenafiIssuer",
					APIVersion: "jetstack.io/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "application-team-b",
				},
				Spec: veiv1alpha1.VenafiCertificateSource{
					Tpp: &veiv1alpha1.TppCertificateIssuer{
						PolicyDn: `\VED\Policy\Teams\ApplicationTeamA`,
						Url:      "https://tpp1.example.com",
					},
				},
			},
		},
		VenafiClusterIssuers: []*veiv1alpha1.VenafiClusterIssuer{
			{
				TypeMeta: metav1.TypeMeta{
					Kind:       "VenafiClusterIssuer",
					APIVersion: "jetstack.io/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "application-team-a",
				},
				Spec: veiv1alpha1.VenafiCertificateSource{
					Tpp: &veiv1alpha1.TppCertificateIssuer{
						PolicyDn: `\VED\Policy\Teams\ApplicationTeamA`,
						Url:      "https://tpp1.example.com",
					},
				},
			},
		},
	}

	for testCaseName, testCase := range testCases {
		t.Run(testCaseName, func(t *testing.T) {
			issuers, err := ExtractOperatorManageableIssuersFromBackupFile(testCase.backupFilePath)
			require.NoError(t, err)

			require.Equal(t, expectedIssuers, issuers)
		})
	}
}
