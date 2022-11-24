package restore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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

	var resources []*unstructured.Unstructured
	// handle both JSON and YAML backup formats
	// JSON backups are formatted as a Kubernetes v1 List, this is so that the
	// file can be applied to the cluster using kubectl apply -f <file>. This
	// does however make the unmarsalling of the file marginally more
	// complicated as we see here.
	if strings.HasSuffix(strings.ToLower(backupFilePath), ".json") {
		rawJSON, err := io.ReadAll(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read backup file: %w", err)
		}

		var list corev1.List
		err = json.Unmarshal(rawJSON, &list)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal backup file as corev1.List: %w", err)
		}

		for _, item := range list.Items {
			decoder := json.NewDecoder(bytes.NewReader(item.Raw))

			var parsedItem unstructured.Unstructured
			err := decoder.Decode(&parsedItem)
			if err != nil {
				return nil, fmt.Errorf("failed to decode item from backup file: %w", err)
			}

			resources = append(resources, &parsedItem)
		}
	} else if strings.HasSuffix(strings.ToLower(backupFilePath), ".yaml") {
		resources, err = yaml.Load(file)
		if err != nil {
			return nil, fmt.Errorf("failed to load backup file: %w", err)
		}
	} else {
		return nil, fmt.Errorf("unsupported backup format for: %q, must be JSON or YAML file", backupFilePath)
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
				return nil, fmt.Errorf("ailed to convert unstructured to cert-manager.io/v1 ClusterIssuer: %w", err)
			}

			restoredIssuers.CertManagerClusterIssuers = append(restoredIssuers.CertManagerClusterIssuers, issuer)
		}
	}

	return &restoredIssuers, nil
}
