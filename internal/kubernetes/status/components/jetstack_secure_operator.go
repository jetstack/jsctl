package components

import (
	"strings"
)

type JetstackSecureOperatorStatus struct {
	namespace, version string
}

func (j *JetstackSecureOperatorStatus) Name() string {
	return "jetstack-secure-operator"
}

func (j *JetstackSecureOperatorStatus) Namespace() string {
	return j.namespace
}

func (j *JetstackSecureOperatorStatus) Version() string {
	return j.version
}

func (j *JetstackSecureOperatorStatus) MarshalYAML() (interface{}, error) {
	return map[string]string{
		"namespace": j.namespace,
		"version":   j.version,
	}, nil
}

func (j *JetstackSecureOperatorStatus) Match(md *MatchData) (bool, error) {
	var found bool

	for _, pod := range md.Pods {
		for _, container := range pod.Spec.Containers {
			if strings.Contains(container.Image, "js-operator:") {
				found = true
				j.namespace = pod.Namespace
				j.version = container.Image[strings.LastIndex(container.Image, ":")+1:]
			}
		}
	}

	return found, nil
}

// NewJetstackSecureOperatorStatus returns an instance that can be used in testing
func NewJetstackSecureOperatorStatus(namespace, version string) *JetstackSecureOperatorStatus {
	return &JetstackSecureOperatorStatus{
		namespace: namespace,
		version:   version,
	}
}
