package components

import (
	"strings"

	v1core "k8s.io/api/core/v1"
)

type CertManagerControllerStatus struct {
	namespace, version string
}

func (c *CertManagerControllerStatus) Name() string {
	return "cert-manager-controller"
}

func (c *CertManagerControllerStatus) Namespace() string {
	return c.namespace
}

func (c *CertManagerControllerStatus) Version() string {
	return c.version
}

func (c *CertManagerControllerStatus) MarshalYAML() (interface{}, error) {
	return map[string]string{
		"namespace": c.namespace,
		"version":   c.version,
	}, nil
}

// NewCertManagerControllerStatus returns an instance that can be used in testing
func NewCertManagerControllerStatus(namespace, version string) *CertManagerControllerStatus {
	return &CertManagerControllerStatus{
		namespace: namespace,
		version:   version,
	}
}

func FindCertManagerController(pod *v1core.Pod) (*CertManagerControllerStatus, error) {
	var status CertManagerControllerStatus
	status.namespace = pod.Namespace

	found := false
	for _, container := range pod.Spec.Containers {
		if strings.Contains(container.Image, "cert-manager-controller") {
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
