package clusters

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jetstack/jsctl/internal/client"
	"github.com/jetstack/jsctl/internal/cluster"
	commandErrors "github.com/jetstack/jsctl/internal/command/errors"
	"github.com/jetstack/jsctl/internal/command/types"
	"github.com/jetstack/jsctl/internal/config"
	"github.com/jetstack/jsctl/internal/kubernetes"
)

// Connect returns a new cobra.Command that connects a cluster to the control plane.
func Connect(run types.RunFunc, kubeConfigPath, apiURL *string, useStdout *bool) *cobra.Command {
	const defaultRegistry = "quay.io/jetstack"
	var registry string

	cmd := &cobra.Command{
		Use:   "connect name",
		Short: "Creates a new cluster in the control plane and deploys the agent in your current kubenetes context",
		Args:  cobra.MatchAll(cobra.ExactArgs(1)),
		Run: run(func(ctx context.Context, args []string) error {
			name := args[0]
			if name == "" {
				return errors.New("you must specify a cluster name")
			}

			cnf, ok := config.FromContext(ctx)
			if !ok || cnf.Organization == "" {
				return commandErrors.ErrNoOrganizationName
			}

			http := client.New(ctx, *apiURL)

			serviceAccount, err := cluster.CreateServiceAccount(ctx, http, cnf.Organization, name)
			if err != nil {
				return fmt.Errorf("failed to create service account: %w", err)
			}

			var applier cluster.Applier
			if *useStdout {
				applier = kubernetes.NewStdOutApplier()
			} else {
				applier, err = kubernetes.NewKubeConfigApplier(*kubeConfigPath)
				if err != nil {
					return err
				}
			}

			err = cluster.ApplyAgentYAML(ctx, applier, cluster.ApplyAgentYAMLOptions{
				Organization:   cnf.Organization,
				Name:           name,
				ServiceAccount: serviceAccount,
				ImageRegistry:  registry,
			})

			if err != nil {
				return fmt.Errorf("failed to generate agent YAML: %w", err)
			}

			fmt.Fprintf(os.Stderr, "Once connected, you can view the cluster in the dashboard:\n"+
				"https://platform.jetstack.io/org/%s/certinventory/cluster/%s\n", cnf.Organization, name)

			return nil
		}),
	}

	flags := cmd.PersistentFlags()
	flags.StringVar(&registry, "registry", defaultRegistry, "Specifies an alternative image registry to use for the agent image")

	return cmd
}
