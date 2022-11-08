package components

import (
	"strings"

	v1core "k8s.io/api/core/v1"
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

// NewSmallStepIssuerStatus returns an instance that can be used in testing
func NewSmallStepIssuerStatus(namespace, version string) *SmallStepIssuerStatus {
	return &SmallStepIssuerStatus{
		namespace: namespace,
		version:   version,
	}
}

func FindSmallStepIssuer(pod *v1core.Pod) (*SmallStepIssuerStatus, error) {
	var status SmallStepIssuerStatus
	status.namespace = pod.Namespace

	found := false
	for _, container := range pod.Spec.Containers {
		if strings.Contains(container.Image, "step-issuer") {
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
