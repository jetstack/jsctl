package components

import (
	"strings"
)

type CertManagerCSIDriverSpiffeApproverStatus struct {
	namespace, version string
}

func (c *CertManagerCSIDriverSpiffeApproverStatus) Name() string {
	return "cert-manager-csi-driver-spiffe-approver"
}

func (c *CertManagerCSIDriverSpiffeApproverStatus) Namespace() string {
	return c.namespace
}

func (c *CertManagerCSIDriverSpiffeApproverStatus) Version() string {
	return c.version
}

func (c *CertManagerCSIDriverSpiffeApproverStatus) MarshalYAML() (interface{}, error) {
	return map[string]string{
		"namespace": c.namespace,
		"version":   c.version,
	}, nil
}

func (c *CertManagerCSIDriverSpiffeApproverStatus) Match(md *MatchData) (bool, error) {
	var found bool

	for _, pod := range md.Pods {
		for _, container := range pod.Spec.Containers {
			if strings.Contains(container.Image, "cert-manager-csi-driver-spiffe-approver") {
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

// NewCertManagerCSIDriverSpifferApproverStatus returns an instance that can be used in testing
func NewCertManagerCSIDriverSpifferApproverStatus(namespace, version string) *CertManagerCSIDriverSpiffeApproverStatus {
	return &CertManagerCSIDriverSpiffeApproverStatus{
		namespace: namespace,
		version:   version,
	}
}
