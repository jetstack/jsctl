package components

import (
	"strings"
)

type SmallStepIssuerStatus struct {
	namespace, version string
}

func (s *SmallStepIssuerStatus) Name() string {
	return "small-step-issuer"
}

func (s *SmallStepIssuerStatus) Namespace() string {
	return s.namespace
}

func (s *SmallStepIssuerStatus) Version() string {
	return s.version
}

func (s *SmallStepIssuerStatus) MarshalYAML() (interface{}, error) {
	return map[string]string{
		"namespace": s.namespace,
		"version":   s.version,
	}, nil
}

func (s *SmallStepIssuerStatus) Match(md *MatchData) (bool, error) {
	var found bool

	for _, pod := range md.Pods {
		for _, container := range pod.Spec.Containers {
			if strings.Contains(container.Image, "step-issuer:") {
				found = true
				s.namespace = pod.Namespace
				s.version = container.Image[strings.LastIndex(container.Image, ":")+1:]
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
