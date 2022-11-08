package components

import (
	"strings"

	v1core "k8s.io/api/core/v1"
)

type CertManagerApproverPolicyEnterpriseStatus struct {
	namespace, version string
}

func (c *CertManagerApproverPolicyEnterpriseStatus) Name() string {
	return "cert-manager-approver-policy-enterprise"
}

func (c *CertManagerApproverPolicyEnterpriseStatus) Namespace() string {
	return c.namespace
}

func (c *CertManagerApproverPolicyEnterpriseStatus) Version() string {
	return c.version
}

func (c *CertManagerApproverPolicyEnterpriseStatus) MarshalYAML() (interface{}, error) {
	return map[string]string{
		"namespace": c.namespace,
		"version":   c.version,
	}, nil
}

func (c *CertManagerApproverPolicyEnterpriseStatus) Match(pod *v1core.Pod) (bool, error) {
	c.namespace = pod.Namespace

	found := false
	for _, container := range pod.Spec.Containers {
		if strings.Contains(container.Image, "approver-policy-enterprise") {
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

// NewCertManagerApproverPolicyEnterpriseStatus returns an instance that can be used in testing
func NewCertManagerApproverPolicyEnterpriseStatus(namespace, version string) *CertManagerApproverPolicyEnterpriseStatus {
	return &CertManagerApproverPolicyEnterpriseStatus{
		namespace: namespace,
		version:   version,
	}
}
