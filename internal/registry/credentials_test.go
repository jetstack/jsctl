package registry

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1core "k8s.io/api/core/v1"

	"github.com/jetstack/jsctl/internal/docker"
)

func TestDockerConfigJSON(t *testing.T) {
	dockerConfig, err := DockerConfigJSON("./testdata/key.json")
	assert.NoError(t, err)

	var actualConfig docker.ConfigJSON
	assert.NoError(t, json.Unmarshal(dockerConfig, &actualConfig))
	assert.NotEmpty(t, actualConfig.Auths)

	actualGCR := actualConfig.Auths["eu.gcr.io"]
	assert.NotEmpty(t, actualGCR.Email)
	assert.NotEmpty(t, actualGCR.Password)
	assert.NotEmpty(t, actualGCR.Auth)
	assert.NotEmpty(t, actualGCR.Username)
}

func TestImagePullSecret(t *testing.T) {
	t.Run("It should load valid credentials and generate a secret", func(t *testing.T) {
		keyData, err := os.ReadFile("./testdata/key.json")
		require.NoError(t, err)

		secret, err := ImagePullSecret(string(keyData))
		assert.NoError(t, err)

		assert.EqualValues(t, "jetstack-secure", secret.Namespace)
		assert.EqualValues(t, "jse-gcr-creds", secret.Name)
		assert.EqualValues(t, v1core.SecretTypeDockerConfigJson, secret.Type)
		assert.NotEmpty(t, secret.Data[v1core.DockerConfigJsonKey])

		var actualConfig docker.ConfigJSON
		assert.NoError(t, json.Unmarshal(secret.Data[v1core.DockerConfigJsonKey], &actualConfig))
		assert.NotEmpty(t, actualConfig.Auths)

		actualGCR := actualConfig.Auths["eu.gcr.io"]
		assert.NotEmpty(t, actualGCR.Email)
		assert.NotEmpty(t, actualGCR.Password)
		assert.NotEmpty(t, actualGCR.Auth)
		assert.NotEmpty(t, actualGCR.Username)
	})
}
