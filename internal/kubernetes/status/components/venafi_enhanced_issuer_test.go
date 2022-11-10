package components

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
)

func TestVenafiEnhancedIssuer(t *testing.T) {
	var err error
	data, err := os.ReadFile("fixtures/venafi-enhanced-issuer.json")
	require.NoError(t, err)

	var pod v1.Pod

	err = json.Unmarshal(data, &pod)
	require.NoError(t, err)

	var status VenafiEnhancedIssuerStatus

	found, err := status.Match(&pod)
	require.NoError(t, err)
	require.True(t, found)

	assert.Equal(t, "venafi-enhanced-issuer", status.Name())
	assert.Equal(t, "example", status.Namespace())
	assert.Equal(t, "v0.1.6", status.Version())
}
