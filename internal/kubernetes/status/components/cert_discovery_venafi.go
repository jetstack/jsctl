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

func (c *CertDiscoveryVenafiStatus) Match(pod *v1core.Pod) (bool, error) {
	c.namespace = pod.Namespace

	found := false
	for _, container := range pod.Spec.Containers {
		if strings.Contains(container.Image, "cert-discovery-venafi") {
			found = true
			if strings.Contains(container.Image, ":") {
				c.version = container.Image[strings.LastIndex(container.Image, ":")+1:]
			} else {
				c.version = "unknown"
			}
		}
	}

	return found, nil
}

// NewCertDiscoveryVenafiStatus returns an instance that can be used in testing
func NewCertDiscoveryVenafiStatus(namespace, version string) *CertDiscoveryVenafiStatus {
	return &CertDiscoveryVenafiStatus{
		namespace: namespace,
		version:   version,
	}
}
