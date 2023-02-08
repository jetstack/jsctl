package registry

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
			// TODO: add the same for US since we now sync images there
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
func ImagePullSecret(keyData string) (*corev1.Secret, error) {
	configJSON, err := DockerConfigJSON(keyData)
	if err != nil {
		return nil, fmt.Errorf("failed to generate docker config: %w", err)
	}

	const (
		secretName = "jse-gcr-creds"
		namespace  = "jetstack-secure"
	)

	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: configJSON,
		},
	}

	return secret, nil
}
