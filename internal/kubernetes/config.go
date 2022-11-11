package kubernetes

import (
	"fmt"
	"os"

	"github.com/mitchellh/go-homedir"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// NewConfig returns a new rest.Config instance based on the kubeconfig path provided. If the path is blank, an in-cluster
// configuration is assumed.
func NewConfig(kubeConfig string) (*rest.Config, error) {
	var config *rest.Config
	var err error
	if kubeConfig != "" {
		kubeConfigPath, err := homedir.Expand(kubeConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to expand kubeconfig path: %w", err)
		}

		_, err = os.Stat(kubeConfigPath)
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("kubeconfig doesn't exist: %w", err)
		} else if err != nil {
			return nil, fmt.Errorf("failed to check kubeconfig path: %w", err)
		}

		config, err = clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	} else {
		config, err = rest.InClusterConfig()
	}

	if err != nil {
		return nil, err
	}

	if config == nil {
		return nil, fmt.Errorf("failed to create config, is your kubeconfig present and configured to connect to a cluster that's still running?")
	}

	return config, nil
}
