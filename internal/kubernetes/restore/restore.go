package restore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	veiv1alpha1 "github.com/jetstack/venafi-enhanced-issuer/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/jetstack/jsctl/internal/kubernetes/yaml"
)

// RestoredIssuers contains the issuers and cluster issuers that were extracted
// from a backup file. Issuers which were unsupported are listed in Missed.
type RestoredIssuers struct {
	CertManagerIssuers        []*cmapi.Issuer
	CertManagerClusterIssuers []*cmapi.ClusterIssuer
	VenafiIssuers             []*veiv1alpha1.VenafiIssuer
	VenafiClusterIssuers      []*veiv1alpha1.VenafiClusterIssuer

	// Missed is a list of issuers that are not supported for restore.
	Missed []string

	// NeedsConversion is a list of issuers that are not supported for restore
	// but could be if converted
	NeedsConversion []string
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
		switch resource.GroupVersionKind().Group {
		case "cert-manager.io":
			if resource.GetAPIVersion() != "cert-manager.io/v1" {
				restoredIssuers.NeedsConversion = append(restoredIssuers.NeedsConversion, fmt.Sprintf("%s/%s", resource.GetKind(), resource.GetName()))
				continue
			}
			switch resource.GroupVersionKind().Kind {
			case "Issuer":
				var issuer *cmapi.Issuer

				err = runtime.DefaultUnstructuredConverter.FromUnstructured(resource.Object, &issuer)
				if err != nil {
					return nil, fmt.Errorf("failed to convert unstructured to cert-manager.io/v1 Issuer: %w", err)
				}

				restoredIssuers.CertManagerIssuers = append(restoredIssuers.CertManagerIssuers, issuer)
			case "ClusterIssuer":
				var issuer *cmapi.ClusterIssuer

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
				restoredIssuers.Missed = append(restoredIssuers.Missed, fmt.Sprintf("%s/%s", resource.GetKind(), resource.GetName()))
			}
		}
	}

	return &restoredIssuers, nil
}
