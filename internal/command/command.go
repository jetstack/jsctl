// Package command contains implementations of individual commands made available via the command-line interface.
package command

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jetstack/jsctl/internal/auth"
	"github.com/jetstack/jsctl/internal/config"
)

var (
	stdout     bool
	kubeConfig string
	apiURL     string
	configDir  string

	errNoOrganizationName = errors.New("You do not have an organization selected, select one using: \n\n\tjsctl config set organization [name]")
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
	flags.BoolVar(&stdout, "stdout", false, "If provided, manifests are written to stdout rather than applied to the current cluster")
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

func run(fn func(ctx context.Context, args []string) error) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		var err error
		ctx := cmd.Context()

		defaultConfigDir, err := config.DefaultConfigDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to determine default user config directory, using current directory\n")
			defaultConfigDir = "."
		}

		// if the user is using configDir defaulting, then we need to check for
		// legacy config directories and migrate them if they exist
		if configDir == defaultConfigDir {
			err := config.MigrateDefaultConfig(configDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to migrate legacy config directory: %s\n", err)
				os.Exit(1)
			}
		}

		// write the configuration directory to the context so that it can be
		// used in subcommands
		ctx = context.WithValue(ctx, config.ContextKey{}, configDir)

		// ensure that the config dir specified exists, this allows other
		// commands to write to sub paths of this directory without concern
		// for the config dir existing
		err = os.MkdirAll(configDir, 0700)
		if err != nil {
			exitf("failed to create config directory: %s", err)
		}

		token, err := auth.LoadOAuthToken(ctx)
		switch {
		case errors.Is(err, auth.ErrNoToken):
			break
		case err != nil:
			exitf("failed to load oauth token: %s", err)
		default:
			ctx = auth.TokenToContext(ctx, token)
		}

		cnf, err := config.Load(ctx)
		switch {
		case errors.Is(err, config.ErrNoConfiguration):
			break
		case err != nil:
			exitf("failed to load configuration: %s", err)
		default:
			ctx = config.ToContext(ctx, cnf)
		}

		if err = fn(ctx, args); err != nil {
			exitf(err.Error())
		}
	}
}

func exitf(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}

func defaultKubeConfig() string {
	const defaultLocation = "~/.kube/config"

	val := os.Getenv("KUBECONFIG")
	if val != "" {
		return val
	}

	return defaultLocation
}
