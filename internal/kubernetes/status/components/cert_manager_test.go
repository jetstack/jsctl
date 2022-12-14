package components

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestCertManager(t *testing.T) {
	var err error
	data, err := os.ReadFile("fixtures/cert-manager.json")
	require.NoError(t, err)

	var pods corev1.PodList

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

	verbosityFlagFound, verbosityFlagValue := status.GetControllerFlagValue("v")
	assert.True(t, verbosityFlagFound)
	assert.Equal(t, "2", verbosityFlagValue)

	controllersFlagFound, controllersFlagValue := status.GetControllerFlagValue("controllers")
	assert.True(t, controllersFlagFound)
	assert.Equal(t, "*,-certificaterequests-approver", controllersFlagValue)
}
