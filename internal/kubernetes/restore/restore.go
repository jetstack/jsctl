package restore

import (
	"fmt"
	"os"
	"strings"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	veiv1alpha1 "github.com/jetstack/venafi-enhanced-issuer/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/jetstack/jsctl/internal/kubernetes/yaml"
)

// RestoredIssuers contains the issuers and cluster issuers that were extracted
// from a backup file. Issuers which were unsupported are listed in MissedIssuers.
type RestoredIssuers struct {
	CertManagerIssuers        []*certmanagerv1.Issuer
	CertManagerClusterIssuers []*certmanagerv1.ClusterIssuer
	VenafiIssuers             []*veiv1alpha1.VenafiIssuer
	VenafiClusterIssuers      []*veiv1alpha1.VenafiClusterIssuer

	MissedIssuers []string
}

func ExtractOperatorManageableIssuersFromBackupFile(backupFilePath string) (*RestoredIssuers, error) {
	var restoredIssuers RestoredIssuers

	file, err := os.Open(backupFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open backup file: %w", err)
	}

	resources, err := yaml.Load(file)
	if err != nil {
		return nil, fmt.Errorf("failed to load backup file: %w", err)
	}

	for _, resource := range resources {

		switch resource.GroupVersionKind().Group {
		case "cert-manager.io":
			switch resource.GroupVersionKind().Kind {
			case "Issuer":
				var issuer *certmanagerv1.Issuer

				err = runtime.DefaultUnstructuredConverter.FromUnstructured(resource.Object, &issuer)
				if err != nil {
					return nil, fmt.Errorf("failed to convert unstructured to cert-manager.io/v1 Issuer: %w", err)
				}

				restoredIssuers.CertManagerIssuers = append(restoredIssuers.CertManagerIssuers, issuer)
			case "ClusterIssuer":
				var issuer *certmanagerv1.ClusterIssuer

				err = runtime.DefaultUnstructuredConverter.FromUnstructured(resource.Object, &issuer)
				if err != nil {
					return nil, fmt.Errorf("ailed to convert unstructured to cert-manager.io/v1 ClusterIssuer: %w", err)
				}

				restoredIssuers.CertManagerClusterIssuers = append(restoredIssuers.CertManagerClusterIssuers, issuer)
			}
		case "jetstack.io":
			switch resource.GroupVersionKind().Kind {
			case "VenafiIssuer":
				var issuer *veiv1alpha1.VenafiIssuer

				err = runtime.DefaultUnstructuredConverter.FromUnstructured(resource.Object, &issuer)
				if err != nil {
					return nil, fmt.Errorf("failed to convert unstructured to Venafi Issuer: %w", err)
				}

				restoredIssuers.VenafiIssuers = append(restoredIssuers.VenafiIssuers, issuer)
			case "VenafiClusterIssuer":
				var issuer *veiv1alpha1.VenafiClusterIssuer

				err = runtime.DefaultUnstructuredConverter.FromUnstructured(resource.Object, &issuer)
				if err != nil {
					return nil, fmt.Errorf("failed to convert unstructured to Venafi Cluster Issuer: %w", err)
				}

				restoredIssuers.VenafiClusterIssuers = append(restoredIssuers.VenafiClusterIssuers, issuer)
			}
		default:
			if strings.Contains(resource.GroupVersionKind().Kind, "Issuer") {
				restoredIssuers.MissedIssuers = append(restoredIssuers.MissedIssuers, fmt.Sprintf("%s/%s", resource.GetKind(), resource.GetName()))
			}
		}
	}

	return &restoredIssuers, nil
}
