package components

import (
	"strings"
)

type VenafiOAuthHelperStatus struct {
	namespace, version string
}

func (v *VenafiOAuthHelperStatus) Name() string {
	return "venafi-oauth-helper"
}

func (v *VenafiOAuthHelperStatus) Namespace() string {
	return v.namespace
}

func (v *VenafiOAuthHelperStatus) Version() string {
	return v.version
}

func (v *VenafiOAuthHelperStatus) MarshalYAML() (interface{}, error) {
	return map[string]string{
		"namespace": v.namespace,
		"version":   v.version,
	}, nil
}

func (v *VenafiOAuthHelperStatus) Match(md *MatchData) (bool, error) {
	var found bool

	for _, pod := range md.Pods {
		for _, container := range pod.Spec.Containers {
			if strings.Contains(container.Image, "venafi-oauth-helper:") {
				found = true
				v.namespace = pod.Namespace
				v.version = container.Image[strings.LastIndex(container.Image, ":")+1:]
			}
		}
	}

	return found, nil
}

// NewVenafiOAuthHelperStatus returns an instance that can be used in testing
func NewVenafiOAuthHelperStatus(namespace, version string) *VenafiOAuthHelperStatus {
	return &VenafiOAuthHelperStatus{
		namespace: namespace,
		version:   version,
	}
}
