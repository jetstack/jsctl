package components

import (
	"strings"

	v1core "k8s.io/api/core/v1"
)

type OriginCAIssuerStatus struct {
	namespace, version string
}

func (c *OriginCAIssuerStatus) Name() string {
	return "origin-ca-issuer"
}

func (c *OriginCAIssuerStatus) Namespace() string {
	return c.namespace
}

func (c *OriginCAIssuerStatus) Version() string {
	return c.version
}

func (c *OriginCAIssuerStatus) MarshalYAML() (interface{}, error) {
	return map[string]string{
		"namespace": c.namespace,
		"version":   c.version,
	}, nil
}

func (c *OriginCAIssuerStatus) Match(pod *v1core.Pod) (bool, error) {
	c.namespace = pod.Namespace

	found := false
	for _, container := range pod.Spec.Containers {
		if strings.Contains(container.Image, "origin-ca-issuer") {
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

// NewOriginCAIssuerStatus returns an instance that can be used in testing
func NewOriginCAIssuerStatus(namespace, version string) *OriginCAIssuerStatus {
	return &OriginCAIssuerStatus{
		namespace: namespace,
		version:   version,
	}
}
