package components

import (
	"strings"
)

// CertManagerStatus is a component status for all cert-manager components
// (controller, cainjector, webhook)
type CertManagerStatus struct {
	namespace string

	controllerVersion, cainjectorVersion, webhookVersion string

	controllerArgs []string
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

func (c *CertManagerStatus) GetControllerFlagValue(flag string) (bool, string) {
	for _, arg := range c.controllerArgs {
		if strings.HasPrefix(arg, "--"+flag) {
			return true, strings.TrimPrefix(arg, "--"+flag+"=")
		}
	}

	return false, ""
}

func (c *CertManagerStatus) Match(md *MatchData) (bool, error) {
	var found bool

	c.controllerVersion = missingComponentString
	c.cainjectorVersion = missingComponentString
	c.webhookVersion = missingComponentString

	for _, pod := range md.Pods {
		for _, container := range pod.Spec.Containers {
			if strings.Contains(container.Image, "cert-manager-controller:") {
				found = true
				c.namespace = pod.Namespace
				c.controllerVersion = container.Image[strings.LastIndex(container.Image, ":")+1:]

				c.controllerArgs = container.Args
			}

			if strings.Contains(container.Image, "cert-manager-cainjector:") {
				found = true
				c.namespace = pod.Namespace
				c.cainjectorVersion = container.Image[strings.LastIndex(container.Image, ":")+1:]
			}

			if strings.Contains(container.Image, "cert-manager-webhook:") {
				found = true
				c.namespace = pod.Namespace
				c.webhookVersion = container.Image[strings.LastIndex(container.Image, ":")+1:]
			}
		}
	}

	return found, nil
}

// NewCertManagerStatus returns an instance that can be used in testing
func NewCertManagerStatus(namespace, version string, args []string) *CertManagerStatus {
	return &CertManagerStatus{
		namespace:         namespace,
		controllerVersion: version,
		webhookVersion:    version,
		cainjectorVersion: version,
		controllerArgs:    args,
	}
}
