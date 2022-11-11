package components

import (
	"strings"
)

type OriginCAIssuerStatus struct {
	namespace, version string
}

func (o *OriginCAIssuerStatus) Name() string {
	return "origin-ca-issuer"
}

func (o *OriginCAIssuerStatus) Namespace() string {
	return o.namespace
}

func (o *OriginCAIssuerStatus) Version() string {
	return o.version
}

func (o *OriginCAIssuerStatus) MarshalYAML() (interface{}, error) {
	return map[string]string{
		"namespace": o.namespace,
		"version":   o.version,
	}, nil
}

func (o *OriginCAIssuerStatus) Match(md *MatchData) (bool, error) {
	var found bool

	for _, pod := range md.Pods {
		for _, container := range pod.Spec.Containers {
			if strings.Contains(container.Image, "origin-ca-issuer") {
				found = true
				o.namespace = pod.Namespace
				if strings.Contains(container.Image, ":") {
					o.version = container.Image[strings.LastIndex(container.Image, ":")+1:]
				} else {
					o.version = "unknown"
				}
			}
		}
	}

	return found, nil
}

// NewOriginCAIssuerStatus returns an instance that can be used in testing
func NewOriginCAIssuerStatus(namespace, version string) *OriginCAIssuerStatus {
	return &OriginCAIssuerStatus{
		namespace: namespace,
		version:   version,
	}
}
