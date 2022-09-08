// Package docker contains common types for docker configuration.
package docker

type (
	// ConfigJSON represents a local docker auth config file
	// for pulling images.
	ConfigJSON struct {
		Auths Config `json:"auths"`
	}

	// Config represents the config file used by the docker CLI.
	// This config that represents the credentials that should be used
	// when pulling images from specific image repositories.
	Config map[string]ConfigEntry

	// ConfigEntry holds the user information that grant the access to docker registry
	ConfigEntry struct {
		Username string `json:"username,omitempty"`
		Password string `json:"password,omitempty"`
		Email    string `json:"email,omitempty"`
		Auth     string `json:"auth,omitempty"`
	}
)
