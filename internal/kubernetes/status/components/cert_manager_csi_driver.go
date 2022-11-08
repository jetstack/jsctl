package components

import (
	"strings"

	v1core "k8s.io/api/core/v1"
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

// NewCertManagerCSIDriverStatus returns an instance that can be used in testing
func NewCertManagerCSIDriverStatus(namespace, version string) *CertManagerCSIDriverStatus {
	return &CertManagerCSIDriverStatus{
		namespace: namespace,
		version:   version,
	}
}

func FindCertManagerCSIDriver(pod *v1core.Pod) (*CertManagerCSIDriverStatus, error) {
	var status CertManagerCSIDriverStatus
	status.namespace = pod.Namespace

	found := false
	for _, container := range pod.Spec.Containers {
		if strings.Contains(container.Image, "cert-manager-csi-driver") {
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
