package components

import (
	"strings"

	v1core "k8s.io/api/core/v1"
)

type VenafiEnhancedIssuerStatus struct {
	namespace, version string
}

func (v *VenafiEnhancedIssuerStatus) Name() string {
	return "venafi-enhanced-issuer"
}

func (v *VenafiEnhancedIssuerStatus) Namespace() string {
	return v.namespace
}

func (v *VenafiEnhancedIssuerStatus) Version() string {
	return v.version
}

func (v *VenafiEnhancedIssuerStatus) MarshalYAML() (interface{}, error) {
	return map[string]string{
		"namespace": v.namespace,
		"version":   v.version,
	}, nil
}

func (v *VenafiEnhancedIssuerStatus) Match(pod *v1core.Pod) (bool, error) {
	v.namespace = pod.Namespace

	found := false
	for _, container := range pod.Spec.Containers {
		if strings.Contains(container.Image, "venafi-enhanced-issuer") {
			found = true
			if strings.Contains(container.Image, ":") {
				v.version = container.Image[strings.LastIndex(container.Image, ":")+1:]
			} else {
				v.version = "unknown"
			}
		}
	}

	return found, nil
}

// NewVenafiEnhancedIssuerStatus returns an instance that can be used in testing
func NewVenafiEnhancedIssuerStatus(namespace, version string) *VenafiEnhancedIssuerStatus {
	return &VenafiEnhancedIssuerStatus{
		namespace: namespace,
		version:   version,
	}
}
