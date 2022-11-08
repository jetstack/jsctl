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

	status, err := FindSmallStepIssuer(&pod)
	require.NoError(t, err)
	require.NotNilf(t, status, "expected status to be not nil")

	assert.Equal(t, "small-step-issuer", status.Name())
	assert.Equal(t, "step-issuer-system", status.Namespace())
	assert.Equal(t, "0.3.0", status.Version())
}
