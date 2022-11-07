package components

import (
	"strings"

	v1core "k8s.io/api/core/v1"
)

type CertManagerCAInjectorStatus struct {
	namespace, version string
}

func (c *CertManagerCAInjectorStatus) Name() string {
	return "cert-manager-cainjector"
}

func (c *CertManagerCAInjectorStatus) Namespace() string {
	return c.namespace
}

func (c *CertManagerCAInjectorStatus) Version() string {
	return c.version
}

func (c *CertManagerCAInjectorStatus) MarshalYAML() (interface{}, error) {
	return map[string]string{
		"namespace": c.namespace,
		"version":   c.version,
	}, nil
}

// NewCertManagerCAInjectorStatus returns an instance that can be used in testing
func NewCertManagerCAInjectorStatus(namespace, version string) *CertManagerCAInjectorStatus {
	return &CertManagerCAInjectorStatus{
		namespace: namespace,
		version:   version,
	}
}

func FindCertManagerCAInjector(pod *v1core.Pod) (*CertManagerCAInjectorStatus, error) {
	var status CertManagerCAInjectorStatus
	status.namespace = pod.Namespace

	found := false
	for _, container := range pod.Spec.Containers {
		if strings.Contains(container.Image, "cert-manager-cainjector") {
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
