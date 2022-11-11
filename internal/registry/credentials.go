package registry

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	v1core "k8s.io/api/core/v1"
	v1meta "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jetstack/jsctl/internal/docker"
)

// DockerConfifJSON returns a valid docker config JSON for the given JSON Google Service Account key data
func DockerConfigJSON(keyData string) ([]byte, error) {
	// When constructing a docker config for GCR, you must use the _json_key username and provide
	// any valid looking email address. Methodology for building this secret was taken from the kubectl
	// create secret command:
	// https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/kubectl/pkg/cmd/create/create_secret_docker.go
	const (
		username = "_json_key"
		email    = "auth@jetstack.io"
	)

	auth := username + ":" + keyData
	config := docker.ConfigJSON{
		Auths: map[string]docker.ConfigEntry{
			"eu.gcr.io": {
				Username: username,
				Password: string(keyData),
				Email:    email,
				Auth:     base64.StdEncoding.EncodeToString([]byte(auth)),
			},
		},
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to encode docker config: %w", err)
	}

	return configJSON, nil
}

// ImagePullSecret returns a Kubernetes Secret resource that can be used to pull images from the Jetstack Secure
// The keyData parameter should contain the JSON Google Service account to use in the secret.
func ImagePullSecret(keyData string) (*v1core.Secret, error) {
	configJSON, err := DockerConfigJSON(keyData)
	if err != nil {
		return nil, fmt.Errorf("failed to generate docker config: %w", err)
	}

	const (
		secretName = "jse-gcr-creds"
		namespace  = "jetstack-secure"
	)

	secret := &v1core.Secret{
		TypeMeta: v1meta.TypeMeta{
			APIVersion: v1core.SchemeGroupVersion.String(),
			Kind:       "Secret",
		},
		ObjectMeta: v1meta.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Type: v1core.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			v1core.DockerConfigJsonKey: configJSON,
		},
	}

	return secret, nil
}
