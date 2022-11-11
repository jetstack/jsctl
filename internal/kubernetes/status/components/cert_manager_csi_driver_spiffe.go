package components

import (
	"strings"
)

type CertManagerCSIDriverSPIFFEStatus struct {
	namespace, version string
}

func (c *CertManagerCSIDriverSPIFFEStatus) Name() string {
	return "cert-manager-csi-driver-spiffe"
}

func (c *CertManagerCSIDriverSPIFFEStatus) Namespace() string {
	return c.namespace
}

func (c *CertManagerCSIDriverSPIFFEStatus) Version() string {
	return c.version
}

func (c *CertManagerCSIDriverSPIFFEStatus) MarshalYAML() (interface{}, error) {
	return map[string]string{
		"namespace": c.namespace,
		"version":   c.version,
	}, nil
}

func (c *CertManagerCSIDriverSPIFFEStatus) Match(md *MatchData) (bool, error) {
	var found bool

	for _, pod := range md.Pods {
		for _, container := range pod.Spec.Containers {
			if strings.Contains(container.Image, "cert-manager-csi-driver-spiffe") {
				found = true
				c.namespace = pod.Namespace
				if strings.Contains(container.Image, ":") {
					c.version = container.Image[strings.LastIndex(container.Image, ":")+1:]
				} else {
					c.version = "unknown"
				}
			}
		}
	}

	return found, nil
}

// NewCertManagerCSIDriverSPIFFEStatus returns an instance that can be used in testing
func NewCertManagerCSIDriverSPIFFEStatus(namespace, version string) *CertManagerCSIDriverSPIFFEStatus {
	return &CertManagerCSIDriverSPIFFEStatus{
		namespace: namespace,
		version:   version,
	}
}
