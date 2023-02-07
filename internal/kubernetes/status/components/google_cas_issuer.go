package components

import (
	"strings"
)

type GoogleCASIssuerStatus struct {
	namespace, version string
}

func (g *GoogleCASIssuerStatus) Name() string {
	return "google-cas-issuer"
}

func (g *GoogleCASIssuerStatus) Namespace() string {
	return g.namespace
}

func (g *GoogleCASIssuerStatus) Version() string {
	return g.version
}

func (g *GoogleCASIssuerStatus) MarshalYAML() (interface{}, error) {
	return map[string]string{
		"namespace": g.namespace,
		"version":   g.version,
	}, nil
}

func (g *GoogleCASIssuerStatus) Match(md *MatchData) (bool, error) {
	var found bool

	for _, pod := range md.Pods {
		for _, container := range pod.Spec.Containers {
			if strings.Contains(container.Image, "google-cas-issuer:") {
				found = true
				g.namespace = pod.Namespace
				g.version = container.Image[strings.LastIndex(container.Image, ":")+1:]
			}
		}
	}

	return found, nil
}

// NewGoogleCASIssuerStatus returns an instance that can be used in testing
func NewGoogleCASIssuerStatus(namespace, version string) *GoogleCASIssuerStatus {
	return &GoogleCASIssuerStatus{
		namespace: namespace,
		version:   version,
	}
}
