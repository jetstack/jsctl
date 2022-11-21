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
	backupFilePath := "fixtures/backup.yaml"

	issuers, err := ExtractOperatorManageableIssuersFromBackupFile(backupFilePath)
	require.NoError(t, err)

	require.Len(t, issuers.CertManagerIssuers, 1)
	require.Len(t, issuers.CertManagerClusterIssuers, 1)

	require.Equal(t, []string{
		"AWSPCAIssuer/pca-sample",
		"GoogleCASIssuer/googlecasissuer-sample",
	}, issuers.MissedIssuers)

	require.Equal(t, []*certmanagerv1.Issuer{
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
	}, issuers.CertManagerIssuers)

	require.Equal(t, []*certmanagerv1.ClusterIssuer{
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
	}, issuers.CertManagerClusterIssuers)
}
