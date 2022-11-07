package components

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
)

func TestJetstackSecureAgent(t *testing.T) {
	var err error
	data, err := os.ReadFile("fixtures/jetstack-secure-agent.json")
	require.NoError(t, err)

	var pod v1.Pod

	err = json.Unmarshal(data, &pod)
	require.NoError(t, err)

	status, err := FindJetstackSecureAgent(&pod)
	require.NoError(t, err)
	require.NotNilf(t, status, "expected status to be not nil")

	assert.Equal(t, "jetstack-secure-agent", status.Name())
	assert.Equal(t, "jetstack-secure", status.Namespace())
	assert.Equal(t, "v0.1.38", status.Version())
}
