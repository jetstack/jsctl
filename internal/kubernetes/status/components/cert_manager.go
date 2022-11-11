package components

import (
	"strings"
)

const (
	missingComponentString = "componentMissing"
	unknownVersionString   = "unknownVersion"
)

// CertManagerStatus is a component status for all cert-manager components
// (controller, cainjector, webhook)
type CertManagerStatus struct {
	namespace string

	controllerVersion, cainjectorVersion, webhookVersion string
}

func (c *CertManagerStatus) Name() string {
	return "cert-manager"
}

func (c *CertManagerStatus) Namespace() string {
	return c.namespace
}

func (c *CertManagerStatus) Version() string {
	return c.controllerVersion
}

func (c *CertManagerStatus) MarshalYAML() (interface{}, error) {
	return map[string]interface{}{
		"namespace": c.namespace,
		"versions": map[string]string{
			"controller": c.controllerVersion,
			"webhook":    c.webhookVersion,
			"cainjector": c.cainjectorVersion,
		},
	}, nil
}

func (c *CertManagerStatus) Match(md *MatchData) (bool, error) {
	var found bool

	c.controllerVersion = missingComponentString
	c.cainjectorVersion = missingComponentString
	c.webhookVersion = missingComponentString

	for _, pod := range md.Pods {
		for _, container := range pod.Spec.Containers {
			if strings.Contains(container.Image, "cert-manager-controller") {
				found = true
				c.namespace = pod.Namespace
				if strings.Contains(container.Image, ":") {
					c.controllerVersion = container.Image[strings.LastIndex(container.Image, ":")+1:]
				} else {
					c.controllerVersion = unknownVersionString
				}
			}

			if strings.Contains(container.Image, "cert-manager-cainjector") {
				found = true
				c.namespace = pod.Namespace
				if strings.Contains(container.Image, ":") {
					c.cainjectorVersion = container.Image[strings.LastIndex(container.Image, ":")+1:]
				} else {
					c.cainjectorVersion = unknownVersionString
				}
			}

			if strings.Contains(container.Image, "cert-manager-webhook") {
				found = true
				c.namespace = pod.Namespace
				if strings.Contains(container.Image, ":") {
					c.webhookVersion = container.Image[strings.LastIndex(container.Image, ":")+1:]
				} else {
					c.webhookVersion = unknownVersionString
				}
			}
		}
	}

	return found, nil
}

// NewCertManagerStatus returns an instance that can be used in testing
func NewCertManagerStatus(namespace, version string) *CertManagerStatus {
	return &CertManagerStatus{
		namespace:         namespace,
		controllerVersion: version,
		webhookVersion:    version,
		cainjectorVersion: version,
	}
}
