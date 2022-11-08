package components

import (
	"strings"

	v1core "k8s.io/api/core/v1"
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

func (c *CertManagerCSIDriverSPIFFEStatus) Match(pod *v1core.Pod) (bool, error) {
	c.namespace = pod.Namespace

	found := false
	for _, container := range pod.Spec.Containers {
		if strings.Contains(container.Image, "cert-manager-csi-driver-spiffe") {
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

// NewCertManagerCSIDriverSPIFFEStatus returns an instance that can be used in testing
func NewCertManagerCSIDriverSPIFFEStatus(namespace, version string) *CertManagerCSIDriverSPIFFEStatus {
	return &CertManagerCSIDriverSPIFFEStatus{
		namespace: namespace,
		version:   version,
	}
}
