package components

import (
	"strings"
)

type SmallStepIssuerStatus struct {
	namespace, version string
}

func (c *SmallStepIssuerStatus) Name() string {
	return "small-step-issuer"
}

func (c *SmallStepIssuerStatus) Namespace() string {
	return c.namespace
}

func (c *SmallStepIssuerStatus) Version() string {
	return c.version
}

func (c *SmallStepIssuerStatus) MarshalYAML() (interface{}, error) {
	return map[string]string{
		"namespace": c.namespace,
		"version":   c.version,
	}, nil
}

func (c *SmallStepIssuerStatus) Match(md *MatchData) (bool, error) {
	var found bool

	for _, pod := range md.Pods {
		for _, container := range pod.Spec.Containers {
			if strings.Contains(container.Image, "step-issuer") {
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

// NewSmallStepIssuerStatus returns an instance that can be used in testing
func NewSmallStepIssuerStatus(namespace, version string) *SmallStepIssuerStatus {
	return &SmallStepIssuerStatus{
		namespace: namespace,
		version:   version,
	}
}
