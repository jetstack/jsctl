package status

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/client-go/rest"

	"github.com/jetstack/jsctl/internal/kubernetes"
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

	crdClient, err := kubernetes.NewCRDClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("error creating CRD client: %w", err)
	}

	groups := []string{
		"cert-manager.io",
		"jetstack.io",
	}

	crds, err := crdClient.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("error querying for CRDs: %w", err)
	}

	for _, g := range groups {
		var crdGroup crdGroup
		crdGroup.Name = g
		for _, crd := range crds {
			if strings.HasSuffix(crd, g) {
				crdGroup.CRDs = append(crdGroup.CRDs, crd)
			}
		}
		status.CRDGroups = append(status.CRDGroups, crdGroup)
	}

	return &status, nil
}
