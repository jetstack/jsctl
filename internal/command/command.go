// Package command contains implementations of individual commands made available via the command-line interface.
package command

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jetstack/jsctl/internal/config"
)

var (
	useStdout  bool
	kubeConfig string
	apiURL     string
	configDir  string
)

// Command returns the root cobra.Command instance for the entire command-line interface.
func Command() *cobra.Command {
	var err error

	cmd := &cobra.Command{
		Use:   "jsctl",
		Short: "Command-line tool for the Jetstack Secure Control Plane",
	}

	// determine the default location of the jsctl config file
	defaultConfigDir, err := config.DefaultConfigDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to determine default user config directory, using current directory")
		defaultConfigDir = "."
	}

	flags := cmd.PersistentFlags()
	flags.BoolVar(&useStdout, "stdout", false, "If provided, manifests are written to stdout rather than applied to the current cluster")
	flags.StringVar(&kubeConfig, "kubeconfig", defaultKubeConfig(), "Location of the user's kubeconfig file for applying directly to the cluster")
	flags.StringVar(&apiURL, "api-url", "https://platform.jetstack.io", "Base URL of the control-plane API")
	flags.StringVar(&configDir, "config", defaultConfigDir, "Base URL of the control-plane API")

	cmd.AddCommand(
		Auth(),
		Clusters(),
		Config(),
		Operator(),
		Organizations(),
		Registry(),
		Users(),
		Version(&cmd.Version),
	)

	return cmd
}

func defaultKubeConfig() string {
	const defaultLocation = "~/.kube/config"

	val := os.Getenv("KUBECONFIG")
	if val != "" {
		return val
	}

	return defaultLocation
}
