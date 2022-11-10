package components

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
)

func TestCertManagerTrustManager(t *testing.T) {
	var err error
	data, err := os.ReadFile("fixtures/cert-manager-trust-manager.json")
	require.NoError(t, err)

	var pod v1.Pod

	err = json.Unmarshal(data, &pod)
	require.NoError(t, err)

	var status CertManagerTrustManagerStatus

	found, err := status.Match(&pod)
	require.NoError(t, err)
	require.True(t, found)

	assert.Equal(t, "trust-manager", status.Name())
	assert.Equal(t, "cert-manager", status.Namespace())
	assert.Equal(t, "v0.3.0", status.Version())
}
