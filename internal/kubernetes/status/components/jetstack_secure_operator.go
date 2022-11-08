package components

import (
	"strings"

	v1core "k8s.io/api/core/v1"
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

func (j *JetstackSecureOperatorStatus) Match(pod *v1core.Pod) (bool, error) {
	j.namespace = pod.Namespace

	found := false
	for _, container := range pod.Spec.Containers {
		if strings.Contains(container.Image, "js-operator") {
			found = true
			if strings.Contains(container.Image, ":") {
				j.version = container.Image[strings.LastIndex(container.Image, ":")+1:]
			} else {
				j.version = "unknown"
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
