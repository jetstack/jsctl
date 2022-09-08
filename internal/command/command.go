// Package command contains implementations of individual commands made available via the command-line interface.
package command

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/jetstack/jsctl/internal/auth"
	"github.com/jetstack/jsctl/internal/config"
	"github.com/spf13/cobra"
)

var (
	stdout     bool
	kubeConfig string
	apiURL     string

	errNoOrganizationName = errors.New("You do not have an organization selected, select one using: \n\n\tjsctl config set organization [name]")
)

// Command returns the root cobra.Command instance for the entire command-line interface.
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "jsctl",
		Short: "Command-line tool for the Jetstack Secure Control Plane",
	}

	flags := cmd.PersistentFlags()
	flags.BoolVar(&stdout, "stdout", false, "If provided, manifests are written to stdout rather than applied to the current cluster")
	flags.StringVar(&kubeConfig, "kubeconfig", defaultKubeConfig(), "Location of the user's kubeconfig file for applying directly to the cluster")
	flags.StringVar(&apiURL, "api-url", "https://platform.jetstack.io", "Base URL of the control-plane API")

	cmd.AddCommand(
		Auth(),
		Clusters(),
		Config(),
		Operator(),
		Organizations(),
		TrustDomains(),
		Users(),
	)

	return cmd
}

func run(fn func(ctx context.Context, args []string) error) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		token, err := auth.LoadOAuthToken()
		switch {
		case errors.Is(err, auth.ErrNoToken):
			break
		case err != nil:
			exitf("failed to load oauth token: %s", err)
		default:
			ctx = auth.TokenToContext(ctx, token)
		}

		cnf, err := config.Load()
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
