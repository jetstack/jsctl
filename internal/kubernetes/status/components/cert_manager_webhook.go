package components

import (
	"strings"
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

func (c *CertManagerWebhookStatus) Match(md *MatchData) (bool, error) {
	var found bool

	for _, pod := range md.Pods {
		for _, container := range pod.Spec.Containers {
			if strings.Contains(container.Image, "cert-manager-webhook") {
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

// NewCertManagerWebhookStatus returns an instance that can be used in testing
func NewCertManagerWebhookStatus(namespace, version string) *CertManagerWebhookStatus {
	return &CertManagerWebhookStatus{
		namespace: namespace,
		version:   version,
	}
}
