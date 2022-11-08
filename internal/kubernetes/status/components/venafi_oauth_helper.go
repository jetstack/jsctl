package components

import (
	"strings"

	v1core "k8s.io/api/core/v1"
)

type VenafiOAuthHelperStatus struct {
	namespace, version string
}

func (c *VenafiOAuthHelperStatus) Name() string {
	return "venafi-oauth-helper"
}

func (c *VenafiOAuthHelperStatus) Namespace() string {
	return c.namespace
}

func (c *VenafiOAuthHelperStatus) Version() string {
	return c.version
}

func (c *VenafiOAuthHelperStatus) MarshalYAML() (interface{}, error) {
	return map[string]string{
		"namespace": c.namespace,
		"version":   c.version,
	}, nil
}

func (c *VenafiOAuthHelperStatus) Match(pod *v1core.Pod) (bool, error) {
	c.namespace = pod.Namespace

	found := false
	for _, container := range pod.Spec.Containers {
		if strings.Contains(container.Image, "venafi-oauth-helper") {
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

// NewVenafiOAuthHelperStatus returns an instance that can be used in testing
func NewVenafiOAuthHelperStatus(namespace, version string) *VenafiOAuthHelperStatus {
	return &VenafiOAuthHelperStatus{
		namespace: namespace,
		version:   version,
	}
}
