package components

import (
	"strings"

	v1core "k8s.io/api/core/v1"
)

type CertManagerWebhookStatus struct {
	namespace, version string
}

func (c *CertManagerWebhookStatus) Name() string {
	return "cert-manager-webhook"
}

func (c *CertManagerWebhookStatus) Namespace() string {
	return c.namespace
}

func (c *CertManagerWebhookStatus) Version() string {
	return c.version
}

func (c *CertManagerWebhookStatus) MarshalYAML() (interface{}, error) {
	return map[string]string{
		"namespace": c.namespace,
		"version":   c.version,
	}, nil
}

// NewCertManagerWebhookStatus returns an instance that can be used in testing
func NewCertManagerWebhookStatus(namespace, version string) *CertManagerWebhookStatus {
	return &CertManagerWebhookStatus{
		namespace: namespace,
		version:   version,
	}
}

func FindCertManagerWebhook(pod *v1core.Pod) (*CertManagerWebhookStatus, error) {
	var status CertManagerWebhookStatus
	status.namespace = pod.Namespace

	found := false
	for _, container := range pod.Spec.Containers {
		if strings.Contains(container.Image, "cert-manager-webhook") {
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