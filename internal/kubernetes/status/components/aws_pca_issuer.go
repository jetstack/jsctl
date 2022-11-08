package components

import (
	"strings"

	v1core "k8s.io/api/core/v1"
)

type AWSPCAIssuerStatus struct {
	namespace, version string
}

func (a *AWSPCAIssuerStatus) Name() string {
	return "aws-pca-issuer"
}

func (a *AWSPCAIssuerStatus) Namespace() string {
	return a.namespace
}

func (a *AWSPCAIssuerStatus) Version() string {
	return a.version
}

func (a *AWSPCAIssuerStatus) MarshalYAML() (interface{}, error) {
	return map[string]string{
		"namespace": a.namespace,
		"version":   a.version,
	}, nil
}

func (a *AWSPCAIssuerStatus) Match(pod *v1core.Pod) (bool, error) {
	a.namespace = pod.Namespace

	found := false
	for _, container := range pod.Spec.Containers {
		if strings.Contains(container.Image, "cert-manager-aws-privateca-issuer") {
			found = true
			if strings.Contains(container.Image, ":") {
				a.version = container.Image[strings.LastIndex(container.Image, ":")+1:]
			} else {
				a.version = "unknown"
			}
		}
	}

	return found, nil
}

// NewAWSPCAIssuerStatus returns an instance that can be used in testing
func NewAWSPCAIssuerStatus(namespace, version string) *AWSPCAIssuerStatus {
	return &AWSPCAIssuerStatus{
		namespace: namespace,
		version:   version,
	}
}
