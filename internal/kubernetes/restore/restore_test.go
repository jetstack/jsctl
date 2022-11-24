package restore

import (
	"testing"

	certmanageracmev1 "github.com/cert-manager/cert-manager/pkg/apis/acme/v1"
	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmanagermetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
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

	expectedIssuers := []*certmanagerv1.Issuer{
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
	}
	expectedClusterIssuers := []*certmanagerv1.ClusterIssuer{
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
	}

	for testCaseName, testCase := range testCases {
		t.Run(testCaseName, func(t *testing.T) {
			issuers, err := ExtractOperatorManageableIssuersFromBackupFile(testCase.backupFilePath)
			require.NoError(t, err)

			require.Len(t, issuers.CertManagerIssuers, 1)
			require.Len(t, issuers.CertManagerClusterIssuers, 1)

			require.Equal(t, []string{
				"AWSPCAIssuer/pca-sample",
				"GoogleCASIssuer/googlecasissuer-sample",
			}, issuers.MissedIssuers)

			require.Equal(t, expectedIssuers, issuers.CertManagerIssuers)
			require.Equal(t, expectedClusterIssuers, issuers.CertManagerClusterIssuers)
		})
	}
}
