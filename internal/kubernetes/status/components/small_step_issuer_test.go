package components

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
)

func TestSmallStepIssuer(t *testing.T) {
	var err error
	data, err := os.ReadFile("fixtures/small-step-issuer.json")
	require.NoError(t, err)

	var pod v1.Pod

	err = json.Unmarshal(data, &pod)
	require.NoError(t, err)

	var status SmallStepIssuerStatus

	md := &MatchData{
		Pods: []v1.Pod{pod},
	}

	found, err := status.Match(md)
	require.NoError(t, err)
	require.True(t, found)

	assert.Equal(t, "small-step-issuer", status.Name())
	assert.Equal(t, "step-issuer-system", status.Namespace())
	assert.Equal(t, "0.3.0", status.Version())
}
