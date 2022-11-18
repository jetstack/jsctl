// Package command contains implementations of individual commands made available via the command-line interface.
package command

import (
	"fmt"
	"os"
	"os/user"
	"strings"

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
	// get the current user to replace ~ in the default config dir, this makes
	// the generation of the docs consistent
	currentUser, err := user.Current()
	if err != nil {
		currentUser.Username = "unknown"
	}

	flags := cmd.PersistentFlags()
	flags.BoolVar(&useStdout, "stdout", false, "If provided, manifests are written to stdout rather than applied to the current cluster")
	flags.StringVar(&kubeConfig, "kubeconfig", defaultKubeConfig(), "Location of the user's kubeconfig file for applying directly to the cluster")
	flags.StringVar(&apiURL, "api-url", "https://platform.jetstack.io", "Base URL of the control-plane API")
	flags.StringVar(
		&configDir,
		"config",
		strings.Replace(defaultConfigDir, currentUser.Username, "USER", 1),
		"Location of the user's jsctl config directory",
	)

	cmd.AddCommand(
		Auth(),
		Clusters(),
		Config(),
		Experimental(),
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
