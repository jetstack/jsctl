package components

import (
	"strings"
)

type KMSIssuerStatus struct {
	namespace, version string
}

func (k *KMSIssuerStatus) Name() string {
	return "kms-issuer"
}

func (k *KMSIssuerStatus) Namespace() string {
	return k.namespace
}

func (k *KMSIssuerStatus) Version() string {
	return k.version
}

func (k *KMSIssuerStatus) MarshalYAML() (interface{}, error) {
	return map[string]string{
		"namespace": k.namespace,
		"version":   k.version,
	}, nil
}

func (k *KMSIssuerStatus) Match(md *MatchData) (bool, error) {
	var found bool

	for _, pod := range md.Pods {
		for _, container := range pod.Spec.Containers {
			if strings.Contains(container.Image, "kms-issuer") {
				found = true
				k.namespace = pod.Namespace
				if strings.Contains(container.Image, ":") {
					k.version = container.Image[strings.LastIndex(container.Image, ":")+1:]
				} else {
					k.version = "unknown"
				}
			}
		}
	}

	return found, nil
}

// NewKMSIssuerStatus returns an instance that can be used in testing
func NewKMSIssuerStatus(namespace, version string) *KMSIssuerStatus {
	return &KMSIssuerStatus{
		namespace: namespace,
		version:   version,
	}
}
