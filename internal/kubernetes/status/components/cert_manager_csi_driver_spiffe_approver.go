package components

import (
	"strings"

	v1core "k8s.io/api/core/v1"
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

func (c *CertManagerCSIDriverSpiffeApproverStatus) Match(pod *v1core.Pod) (bool, error) {
	c.namespace = pod.Namespace

	found := false
	for _, container := range pod.Spec.Containers {
		if strings.Contains(container.Image, "cert-manager-csi-driver-spiffe-approver") {
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

// NewCertManagerCSIDriverSpifferApproverStatus returns an instance that can be used in testing
func NewCertManagerCSIDriverSpifferApproverStatus(namespace, version string) *CertManagerCSIDriverSpiffeApproverStatus {
	return &CertManagerCSIDriverSpiffeApproverStatus{
		namespace: namespace,
		version:   version,
	}
}
