package components

import (
	"strings"
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

func (v *VenafiEnhancedIssuerStatus) Match(md *MatchData) (bool, error) {
	var found bool

	for _, pod := range md.Pods {
		for _, container := range pod.Spec.Containers {
			if strings.Contains(container.Image, "venafi-enhanced-issuer") {
				found = true
				v.namespace = pod.Namespace
				if strings.Contains(container.Image, ":") {
					v.version = container.Image[strings.LastIndex(container.Image, ":")+1:]
				} else {
					v.version = "unknown"
				}
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
