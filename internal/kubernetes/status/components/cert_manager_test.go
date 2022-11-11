package components

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
)

func TestCertManager(t *testing.T) {
	var err error
	data, err := os.ReadFile("fixtures/cert-manager.json")
	require.NoError(t, err)

	var pods v1.PodList

	err = json.Unmarshal(data, &pods)
	require.NoError(t, err)

	var status CertManagerStatus

	md := &MatchData{
		Pods: pods.Items,
	}

	found, err := status.Match(md)
	require.NoError(t, err)
	require.True(t, found)

	assert.Equal(t, "cert-manager", status.Name())
	assert.Equal(t, "jetstack-secure", status.Namespace())
	assert.Equal(t, "v1.9.1", status.Version())
}
