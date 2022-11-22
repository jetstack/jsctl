package restore

import (
	"fmt"
	"os"
	"strings"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/jetstack/jsctl/internal/kubernetes/yaml"
)

// RestoredIssuers contains the issuers and cluster issuers that were extracted
// from a backup file. Issuers which were unsupported are listed in MissedIssuers.
type RestoredIssuers struct {
	CertManagerIssuers        []*certmanagerv1.Issuer
	CertManagerClusterIssuers []*certmanagerv1.ClusterIssuer

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
		if resource.GetAPIVersion() != "cert-manager.io/v1" {
			if strings.Contains(resource.GetKind(), "Issuer") {
				restoredIssuers.MissedIssuers = append(restoredIssuers.MissedIssuers, fmt.Sprintf("%s/%s", resource.GetKind(), resource.GetName()))
			}
			continue
		}

		if resource.GetKind() == "Issuer" {
			var issuer *certmanagerv1.Issuer

			err = runtime.DefaultUnstructuredConverter.FromUnstructured(resource.Object, &issuer)
			if err != nil {
				return nil, fmt.Errorf("failed to convert unstructured to cert-manager.io/v1 Issuer: %w", err)
			}

			restoredIssuers.CertManagerIssuers = append(restoredIssuers.CertManagerIssuers, issuer)
		}

		if resource.GetKind() == "ClusterIssuer" {
			var issuer *certmanagerv1.ClusterIssuer

			err = runtime.DefaultUnstructuredConverter.FromUnstructured(resource.Object, &issuer)
			if err != nil {
				return nil, fmt.Errorf("failed to convert unstructured to ClusterIssuer: %w", err)
			}

			restoredIssuers.CertManagerClusterIssuers = append(restoredIssuers.CertManagerClusterIssuers, issuer)
		}
	}

	return &restoredIssuers, nil
}
