package components

import (
	"strings"
)

type CertManagerCSIDriverStatus struct {
	namespace, version string
}

func (c *CertManagerCSIDriverStatus) Name() string {
	return "cert-manager-csi-driver"
}

func (c *CertManagerCSIDriverStatus) Namespace() string {
	return c.namespace
}

func (c *CertManagerCSIDriverStatus) Version() string {
	return c.version
}

func (c *CertManagerCSIDriverStatus) MarshalYAML() (interface{}, error) {
	return map[string]string{
		"namespace": c.namespace,
		"version":   c.version,
	}, nil
}

func (c *CertManagerCSIDriverStatus) Match(md *MatchData) (bool, error) {
	var found bool

	for _, pod := range md.Pods {
		for _, container := range pod.Spec.Containers {
			if strings.Contains(container.Image, "cert-manager-csi-driver:") {
				found = true
				c.namespace = pod.Namespace
				c.version = container.Image[strings.LastIndex(container.Image, ":")+1:]
			}
		}
	}

	return found, nil
}

// NewCertManagerCSIDriverStatus returns an instance that can be used in testing
func NewCertManagerCSIDriverStatus(namespace, version string) *CertManagerCSIDriverStatus {
	return &CertManagerCSIDriverStatus{
		namespace: namespace,
		version:   version,
	}
}
