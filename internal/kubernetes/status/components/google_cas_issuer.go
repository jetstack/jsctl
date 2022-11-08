package components

import (
	"strings"

	v1core "k8s.io/api/core/v1"
)

type GoogleCASIssuerStatus struct {
	namespace, version string
}

func (c *GoogleCASIssuerStatus) Name() string {
	return "google-cas-issuer"
}

func (c *GoogleCASIssuerStatus) Namespace() string {
	return c.namespace
}

func (c *GoogleCASIssuerStatus) Version() string {
	return c.version
}

func (c *GoogleCASIssuerStatus) MarshalYAML() (interface{}, error) {
	return map[string]string{
		"namespace": c.namespace,
		"version":   c.version,
	}, nil
}

// NewGoogleCASIssuerStatus returns an instance that can be used in testing
func NewGoogleCASIssuerStatus(namespace, version string) *GoogleCASIssuerStatus {
	return &GoogleCASIssuerStatus{
		namespace: namespace,
		version:   version,
	}
}

func FindGoogleCASIssuer(pod *v1core.Pod) (*GoogleCASIssuerStatus, error) {
	var status GoogleCASIssuerStatus
	status.namespace = pod.Namespace

	found := false
	for _, container := range pod.Spec.Containers {
		if strings.Contains(container.Image, "google-cas-issuer") {
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
