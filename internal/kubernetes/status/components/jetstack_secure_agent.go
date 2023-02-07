package components

import (
	"strings"
)

type JetstackSecureAgentStatus struct {
	namespace, version string
}

func (j *JetstackSecureAgentStatus) Name() string {
	return "jetstack-secure-agent"
}

func (j *JetstackSecureAgentStatus) Namespace() string {
	return j.namespace
}

func (j *JetstackSecureAgentStatus) Version() string {
	return j.version
}

func (j *JetstackSecureAgentStatus) MarshalYAML() (interface{}, error) {
	return map[string]string{
		"namespace": j.namespace,
		"version":   j.version,
	}, nil
}

func (j *JetstackSecureAgentStatus) Match(md *MatchData) (bool, error) {
	var found bool

	for _, pod := range md.Pods {
		for _, container := range pod.Spec.Containers {
			if strings.Contains(container.Image, "jetstack/preflight:") {
				found = true
				j.namespace = pod.Namespace
				j.version = container.Image[strings.LastIndex(container.Image, ":")+1:]
			}
		}
	}

	return found, nil
}

// NewJetstackSecureAgentStatus returns an instance that can be used in testing
func NewJetstackSecureAgentStatus(namespace, version string) *JetstackSecureAgentStatus {
	return &JetstackSecureAgentStatus{
		namespace: namespace,
		version:   version,
	}
}
