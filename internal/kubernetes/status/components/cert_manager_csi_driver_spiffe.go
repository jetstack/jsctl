package components

import (
	"strings"
)

type CertManagerCSIDriverSPIFFEStatus struct {
	namespace                         string
	csiDriverVersion, approverVersion string
}

func (c *CertManagerCSIDriverSPIFFEStatus) Name() string {
	return "cert-manager-csi-driver-spiffe"
}

func (c *CertManagerCSIDriverSPIFFEStatus) Namespace() string {
	return c.namespace
}

func (c *CertManagerCSIDriverSPIFFEStatus) Version() string {
	return c.csiDriverVersion
}

func (c *CertManagerCSIDriverSPIFFEStatus) MarshalYAML() (interface{}, error) {
	return map[string]interface{}{
		"namespace": c.namespace,
		"versions": map[string]string{
			"csi-driver": c.csiDriverVersion,
			"approver":   c.approverVersion,
		},
	}, nil
}

func (c *CertManagerCSIDriverSPIFFEStatus) Match(md *MatchData) (bool, error) {
	var found bool

	c.csiDriverVersion = missingComponentString
	c.approverVersion = missingComponentString

	for _, pod := range md.Pods {
		for _, container := range pod.Spec.Containers {
			if strings.Contains(container.Image, "cert-manager-csi-driver-spiffe") {
				found = true
				c.namespace = pod.Namespace
				if strings.Contains(container.Image, ":") {
					c.csiDriverVersion = container.Image[strings.LastIndex(container.Image, ":")+1:]
				} else {
					c.csiDriverVersion = "unknown"
				}
			}

			if strings.Contains(container.Image, "cert-manager-csi-driver-spiffe-approver") {
				found = true
				c.namespace = pod.Namespace
				if strings.Contains(container.Image, ":") {
					c.approverVersion = container.Image[strings.LastIndex(container.Image, ":")+1:]
				} else {
					c.approverVersion = "unknown"
				}
			}
		}
	}

	return found, nil
}

// NewCertManagerCSIDriverSPIFFEStatus returns an instance that can be used in testing
func NewCertManagerCSIDriverSPIFFEStatus(namespace, version string) *CertManagerCSIDriverSPIFFEStatus {
	return &CertManagerCSIDriverSPIFFEStatus{
		namespace:        namespace,
		approverVersion:  version,
		csiDriverVersion: version,
	}
}
