package components

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
)

func TestCertManagerWebhook(t *testing.T) {
	var err error
	data, err := os.ReadFile("fixtures/cert-manager-webhook.json")
	require.NoError(t, err)

	var pod v1.Pod

	err = json.Unmarshal(data, &pod)
	require.NoError(t, err)

	var status CertManagerWebhookStatus
	found, err := status.Match(&pod)
	require.NoError(t, err)
	require.True(t, found)

	assert.Equal(t, "cert-manager-webhook", status.Name())
	assert.Equal(t, "jetstack-secure", status.Namespace())
	assert.Equal(t, "v1.9.1", status.Version())
}
