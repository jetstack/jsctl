package components

import (
	"strings"

	v1core "k8s.io/api/core/v1"
)

type KMSIssuerStatus struct {
	namespace, version string
}

func (c *KMSIssuerStatus) Name() string {
	return "kms-issuer"
}

func (c *KMSIssuerStatus) Namespace() string {
	return c.namespace
}

func (c *KMSIssuerStatus) Version() string {
	return c.version
}

func (c *KMSIssuerStatus) MarshalYAML() (interface{}, error) {
	return map[string]string{
		"namespace": c.namespace,
		"version":   c.version,
	}, nil
}

// NewKMSIssuerStatus returns an instance that can be used in testing
func NewKMSIssuerStatus(namespace, version string) *KMSIssuerStatus {
	return &KMSIssuerStatus{
		namespace: namespace,
		version:   version,
	}
}

func FindKMSIssuer(pod *v1core.Pod) (*KMSIssuerStatus, error) {
	var status KMSIssuerStatus
	status.namespace = pod.Namespace

	found := false
	for _, container := range pod.Spec.Containers {
		if strings.Contains(container.Image, "kms-issuer") {
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
