package components

import (
	"strings"
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

func (c *CertManagerIstioCSRStatus) Match(md *MatchData) (bool, error) {
	var found bool

	for _, pod := range md.Pods {
		for _, container := range pod.Spec.Containers {
			if strings.Contains(container.Image, "cert-manager-istio-csr:") {
				found = true
				c.namespace = pod.Namespace
				c.version = container.Image[strings.LastIndex(container.Image, ":")+1:]
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
