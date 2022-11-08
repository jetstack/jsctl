package components

import (
	"strings"

	v1core "k8s.io/api/core/v1"
)

type CertDiscoveryVenafiStatus struct {
	namespace, version string
}

func (c *CertDiscoveryVenafiStatus) Name() string {
	return "cert-discovery-venafi"
}

func (c *CertDiscoveryVenafiStatus) Namespace() string {
	return c.namespace
}

func (c *CertDiscoveryVenafiStatus) Version() string {
	return c.version
}

func (c *CertDiscoveryVenafiStatus) MarshalYAML() (interface{}, error) {
	return map[string]string{
		"namespace": c.namespace,
		"version":   c.version,
	}, nil
}

// NewCertDiscoveryVenafiStatus returns an instance that can be used in testing
func NewCertDiscoveryVenafiStatus(namespace, version string) *CertDiscoveryVenafiStatus {
	return &CertDiscoveryVenafiStatus{
		namespace: namespace,
		version:   version,
	}
}

func FindCertDiscoveryVenafi(pod *v1core.Pod) (*CertDiscoveryVenafiStatus, error) {
	var status CertDiscoveryVenafiStatus
	status.namespace = pod.Namespace

	found := false
	for _, container := range pod.Spec.Containers {
		if strings.Contains(container.Image, "cert-discovery-venafi") {
			found = true
			if strings.Contains(container.Image, ":") {
				status.version = container.Image[strings.LastIndex(container.Image, ":")+1:]
			} else {
				status.version = "unknown"
			}
		}
	}

	if found {
		return &status, nil
	}

	return nil, nil
}
