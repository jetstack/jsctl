package components

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
)

func TestIsolatedIssuer(t *testing.T) {
	var err error
	data, err := os.ReadFile("fixtures/isolated-issuer.json")
	require.NoError(t, err)

	var pod v1.Pod

	err = json.Unmarshal(data, &pod)
	require.NoError(t, err)

	var status IsolatedIssuerStatus

	md := &MatchData{
		Pods: []v1.Pod{pod},
	}

	found, err := status.Match(md)
	require.NoError(t, err)
	require.True(t, found)

	assert.Equal(t, "isolated-issuer", status.Name())
	assert.Equal(t, "example", status.Namespace())
	assert.Equal(t, "v0.2.1", status.Version())
}
