package components

import (
	"strings"

	v1core "k8s.io/api/core/v1"
)

type CertManagerIstioCSRStatus struct {
	namespace, version string
}

func (c *CertManagerIstioCSRStatus) Name() string {
	return "istio-csr"
}

func (c *CertManagerIstioCSRStatus) Namespace() string {
	return c.namespace
}

func (c *CertManagerIstioCSRStatus) Version() string {
	return c.version
}

func (c *CertManagerIstioCSRStatus) MarshalYAML() (interface{}, error) {
	return map[string]string{
		"namespace": c.namespace,
		"version":   c.version,
	}, nil
}

func (c *CertManagerIstioCSRStatus) Match(pod *v1core.Pod) (bool, error) {
	c.namespace = pod.Namespace

	found := false
	for _, container := range pod.Spec.Containers {
		if strings.Contains(container.Image, "cert-manager-istio-csr") {
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

// NewCertManagerIstioCSRStatus returns an instance that can be used in testing
func NewCertManagerIstioCSRStatus(namespace, version string) *CertManagerIstioCSRStatus {
	return &CertManagerIstioCSRStatus{
		namespace: namespace,
		version:   version,
	}
}
