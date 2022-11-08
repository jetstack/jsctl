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

// NewAWSPCAIssuerStatus returns an instance that can be used in testing
func NewAWSPCAIssuerStatus(namespace, version string) *AWSPCAIssuerStatus {
	return &AWSPCAIssuerStatus{
		namespace: namespace,
		version:   version,
	}
}

func FindAWSPCAIssuer(pod *v1core.Pod) (*AWSPCAIssuerStatus, error) {
	var status AWSPCAIssuerStatus
	status.namespace = pod.Namespace

	found := false
	for _, container := range pod.Spec.Containers {
		if strings.Contains(container.Image, "cert-manager-aws-privateca-issuer") {
			found = true
			if strings.Contains(container.Image, ":") {
				status.version = container.Image[strings.LastIndex(container.Image, ":")+1:]
			} else {
				status.version = "unknown"
			}
		}
	}

	if found {
		return &status, nil
	}

	return nil, nil
}
