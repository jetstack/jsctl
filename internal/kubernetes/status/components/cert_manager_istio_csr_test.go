package components

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
)

func TestCertManagerIstioCSRIssuer(t *testing.T) {
	var err error
	data, err := os.ReadFile("fixtures/cert-manager-istio-csr.json")
	require.NoError(t, err)

	var pod v1.Pod

	err = json.Unmarshal(data, &pod)
	require.NoError(t, err)

	var status CertManagerIstioCSRStatus

	md := &MatchData{
		Pods: []v1.Pod{pod},
	}

	found, err := status.Match(md)
	require.NoError(t, err)
	require.True(t, found)

	assert.Equal(t, "istio-csr", status.Name())
	assert.Equal(t, "cert-manager", status.Namespace())
	assert.Equal(t, "v0.5.0", status.Version())
}
