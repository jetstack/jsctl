package components

import (
	"strings"

	v1core "k8s.io/api/core/v1"
)

type CertManagerCSIDriverSpifferApproverStatus struct {
	namespace, version string
}

func (c *CertManagerCSIDriverSpifferApproverStatus) Name() string {
	return "cert-manager-csi-driver-spiffe-approver"
}

func (c *CertManagerCSIDriverSpifferApproverStatus) Namespace() string {
	return c.namespace
}

func (c *CertManagerCSIDriverSpifferApproverStatus) Version() string {
	return c.version
}

func (c *CertManagerCSIDriverSpifferApproverStatus) MarshalYAML() (interface{}, error) {
	return map[string]string{
		"namespace": c.namespace,
		"version":   c.version,
	}, nil
}

// NewCertManagerCSIDriverSpifferApproverStatus returns an instance that can be used in testing
func NewCertManagerCSIDriverSpifferApproverStatus(namespace, version string) *CertManagerCSIDriverSpifferApproverStatus {
	return &CertManagerCSIDriverSpifferApproverStatus{
		namespace: namespace,
		version:   version,
	}
}

func FindCertManagerCSIDriverSpifferApprover(pod *v1core.Pod) (*CertManagerCSIDriverSpifferApproverStatus, error) {
	var status CertManagerCSIDriverSpifferApproverStatus
	status.namespace = pod.Namespace

	found := false
	for _, container := range pod.Spec.Containers {
		if strings.Contains(container.Image, "cert-manager-csi-driver-spiffe-approver") {
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
