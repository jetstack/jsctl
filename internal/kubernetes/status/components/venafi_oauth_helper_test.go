package components

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestVenafiOAuthHelper(t *testing.T) {
	var err error
	data, err := os.ReadFile("fixtures/venafi-oauth-helper.json")
	require.NoError(t, err)

	var pod corev1.Pod

	err = json.Unmarshal(data, &pod)
	require.NoError(t, err)

	var status VenafiOAuthHelperStatus

	md := &MatchData{
		Pods: []corev1.Pod{pod},
	}

	found, err := status.Match(md)
	require.NoError(t, err)
	require.True(t, found)

	assert.Equal(t, "venafi-oauth-helper", status.Name())
	assert.Equal(t, "example", status.Namespace())
	assert.Equal(t, "v0.0.0", status.Version())
}
