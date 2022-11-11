package components

import (
	"strings"
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

func (c *CertManagerApproverPolicyStatus) Match(md *MatchData) (bool, error) {
	var found bool

	for _, pod := range md.Pods {
		for _, container := range pod.Spec.Containers {
			if strings.Contains(container.Image, "cert-manager-approver-policy") {
				found = true
				c.namespace = pod.Namespace
				if strings.Contains(container.Image, ":") {
					c.version = container.Image[strings.LastIndex(container.Image, ":")+1:]
				} else {
					c.version = "unknown"
				}
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
