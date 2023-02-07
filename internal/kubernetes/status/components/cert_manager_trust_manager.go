package components

import (
	"strings"
)

type CertManagerTrustManagerStatus struct {
	namespace, version string
}

func (c *CertManagerTrustManagerStatus) Name() string {
	return "trust-manager"
}

func (c *CertManagerTrustManagerStatus) Namespace() string {
	return c.namespace
}

func (c *CertManagerTrustManagerStatus) Version() string {
	return c.version
}

func (c *CertManagerTrustManagerStatus) MarshalYAML() (interface{}, error) {
	return map[string]string{
		"namespace": c.namespace,
		"version":   c.version,
	}, nil
}

func (c *CertManagerTrustManagerStatus) Match(md *MatchData) (bool, error) {
	var found bool

	for _, pod := range md.Pods {
		for _, container := range pod.Spec.Containers {
			if strings.Contains(container.Image, "trust-manager:") {
				found = true
				c.namespace = pod.Namespace
				c.version = container.Image[strings.LastIndex(container.Image, ":")+1:]
			}
		}
	}

	return found, nil
}

// NewCertManagerTrustManagerStatus returns an instance that can be used in testing
func NewCertManagerTrustManagerStatus(namespace, version string) *CertManagerTrustManagerStatus {
	return &CertManagerTrustManagerStatus{
		namespace: namespace,
		version:   version,
	}
}
