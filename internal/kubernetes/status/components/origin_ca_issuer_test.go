package components

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestOriginCAIssuer(t *testing.T) {
	var err error
	data, err := os.ReadFile("fixtures/origin-ca-issuer.json")
	require.NoError(t, err)

	var pod corev1.Pod

	err = json.Unmarshal(data, &pod)
	require.NoError(t, err)

	var status OriginCAIssuerStatus

	md := &MatchData{
		Pods: []corev1.Pod{pod},
	}

	found, err := status.Match(md)
	require.NoError(t, err)
	require.True(t, found)

	assert.Equal(t, "origin-ca-issuer", status.Name())
	assert.Equal(t, "example", status.Namespace())
	assert.Equal(t, "v0.6.1", status.Version())
}
