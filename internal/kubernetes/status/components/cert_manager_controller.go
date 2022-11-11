package components

import (
	"strings"
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

func (c *CertManagerControllerStatus) Match(md *MatchData) (bool, error) {
	var found bool

	for _, pod := range md.Pods {
		for _, container := range pod.Spec.Containers {
			if strings.Contains(container.Image, "cert-manager-controller") {
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

// NewCertManagerControllerStatus returns an instance that can be used in testing
func NewCertManagerControllerStatus(namespace, version string) *CertManagerControllerStatus {
	return &CertManagerControllerStatus{
		namespace: namespace,
		version:   version,
	}
}
