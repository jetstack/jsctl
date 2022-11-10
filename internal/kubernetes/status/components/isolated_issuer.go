package components

import (
	"strings"

	v1core "k8s.io/api/core/v1"
)

type IsolatedIssuerStatus struct {
	namespace, version string
}

func (i *IsolatedIssuerStatus) Name() string {
	return "isolated-issuer"
}

func (i *IsolatedIssuerStatus) Namespace() string {
	return i.namespace
}

func (i *IsolatedIssuerStatus) Version() string {
	return i.version
}

func (i *IsolatedIssuerStatus) MarshalYAML() (interface{}, error) {
	return map[string]string{
		"namespace": i.namespace,
		"version":   i.version,
	}, nil
}

func (i *IsolatedIssuerStatus) Match(pod *v1core.Pod) (bool, error) {
	i.namespace = pod.Namespace

	found := false
	for _, container := range pod.Spec.Containers {
		if strings.Contains(container.Image, "isolated-issuer") {
			found = true
			if strings.Contains(container.Image, ":") {
				i.version = container.Image[strings.LastIndex(container.Image, ":")+1:]
			} else {
				i.version = "unknown"
			}
		}
	}

	return found, nil
}

// NewIsolatedIssuerStatus returns an instance that can be used in testing
func NewIsolatedIssuerStatus(namespace, version string) *IsolatedIssuerStatus {
	return &IsolatedIssuerStatus{
		namespace: namespace,
		version:   version,
	}
}
