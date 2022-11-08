package components

import (
	"strings"

	v1core "k8s.io/api/core/v1"
)

type CertManagerApproverPolicyStatus struct {
	namespace, version string
}

func (c *CertManagerApproverPolicyStatus) Name() string {
	return "cert-manager-approver-policy"
}

func (c *CertManagerApproverPolicyStatus) Namespace() string {
	return c.namespace
}

func (c *CertManagerApproverPolicyStatus) Version() string {
	return c.version
}

func (c *CertManagerApproverPolicyStatus) MarshalYAML() (interface{}, error) {
	return map[string]string{
		"namespace": c.namespace,
		"version":   c.version,
	}, nil
}

func (c *CertManagerApproverPolicyStatus) Match(pod *v1core.Pod) (bool, error) {
	c.namespace = pod.Namespace

	found := false
	for _, container := range pod.Spec.Containers {
		if strings.Contains(container.Image, "cert-manager-approver-policy") {
			found = true
			if strings.Contains(container.Image, ":") {
				c.version = container.Image[strings.LastIndex(container.Image, ":")+1:]
			} else {
				c.version = "unknown"
			}
		}
	}

	return found, nil
}

// NewCertManagerApproverPolicyStatus returns an instance that can be used in testing
func NewCertManagerApproverPolicyStatus(namespace, version string) *CertManagerApproverPolicyStatus {
	return &CertManagerApproverPolicyStatus{
		namespace: namespace,
		version:   version,
	}
}
