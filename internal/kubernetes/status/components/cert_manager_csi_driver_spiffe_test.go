package components

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1core "k8s.io/api/core/v1"
)

func TestCertManagerCSIDriverSPIFFE(t *testing.T) {
	var err error
	data, err := os.ReadFile("fixtures/cert-manager-csi-driver-spiffe.json")
	require.NoError(t, err)

	var pods v1core.PodList

	err = json.Unmarshal(data, &pods)
	require.NoError(t, err)

	var status CertManagerCSIDriverSPIFFEStatus

	md := &MatchData{
		Pods: pods.Items,
	}

	found, err := status.Match(md)
	require.NoError(t, err)
	require.True(t, found)

	assert.Equal(t, "cert-manager-csi-driver-spiffe", status.Name())
	assert.Equal(t, "example", status.Namespace())
	assert.Equal(t, "v0.2.0", status.Version())
}
