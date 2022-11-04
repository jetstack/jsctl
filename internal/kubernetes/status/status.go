package status

import (
	"context"
	"fmt"
	"strings"

	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/client-go/rest"

	"github.com/jetstack/jsctl/internal/kubernetes/clients"
)

// ClusterPreInstallStatus is a collection of information about a cluster that
// can be helpful for users about to install.
type ClusterPreInstallStatus struct {
	CRDGroups []crdGroup `yaml:"crds"`
}

// crdGroup is a list of custom resource definitions that are all part of the
// same group, e.g. cert-manager.io or jetstack.io.
type crdGroup struct {
	Name string
	CRDs []string `yaml:"items"`
}

// GatherClusterPreInstallStatus returns a ClusterPreInstallStatus for the
// supplied cluster
func GatherClusterPreInstallStatus(ctx context.Context, cfg *rest.Config) (*ClusterPreInstallStatus, error) {
	var status ClusterPreInstallStatus

	crdClient, err := clients.NewCRDClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("error creating CRD client: %w", err)
	}

	groups := []string{
		"cert-manager.io",
		"jetstack.io",
	}

	var crdList v1.CustomResourceDefinitionList

	err = crdClient.List(ctx, &crdList)
	if err != nil {
		return nil, fmt.Errorf("error querying for CRDs: %w", err)
	}

	for _, g := range groups {
		var crdGroup crdGroup
		crdGroup.Name = g
		for _, crd := range crdList.Items {
			if strings.HasSuffix(crd.Name, g) {
				crdGroup.CRDs = append(crdGroup.CRDs, crd.Name)
			}
		}
		status.CRDGroups = append(status.CRDGroups, crdGroup)
	}

	return &status, nil
}
