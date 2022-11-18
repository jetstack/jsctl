package clusters

import (
	"context"
	errors2 "errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jetstack/jsctl/internal/client"
	"github.com/jetstack/jsctl/internal/cluster"
	"github.com/jetstack/jsctl/internal/command/errors"
	"github.com/jetstack/jsctl/internal/command/types"
	"github.com/jetstack/jsctl/internal/config"
	"github.com/jetstack/jsctl/internal/prompt"
)

// Delete returns a new cobra.Command for deleting a cluster in the JSCP api
func Delete(run types.RunFunc, apiURL *string) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete [name]",
		Short: "Deletes a cluster from the organization",
		Args:  cobra.ExactValidArgs(1),
		Run: run(func(ctx context.Context, args []string) error {
			cnf, ok := config.FromContext(ctx)
			if !ok || cnf.Organization == "" {
				return errors.ErrNoOrganizationName
			}

			http := client.New(ctx, *apiURL)
			name := args[0]
			if name == "" {
				return errors2.New("you must specify a cluster name")
			}

			if !force {
				ok, err := prompt.YesNo(os.Stdin, os.Stdout, "Are you sure you want to delete cluster %s from organization %s?", name, cnf.Organization)
				switch {
				case err != nil:
					return fmt.Errorf("failed to prompt: %w", err)
				case !ok:
					return nil
				}
			}

			err := cluster.Delete(ctx, http, cnf.Organization, name)
			switch {
			case errors2.Is(err, cluster.ErrNoCluster):
				return fmt.Errorf("cluster %s does not exist in organization %s", name, cnf.Organization)
			case err != nil:
				return fmt.Errorf("failed to delete cluster: %w", err)
			}

			fmt.Printf("Cluster %s was successfully deleted\n", name)
			return nil
		}),
	}

	flags := cmd.PersistentFlags()
	flags.BoolVar(&force, "force", false, "Do not prompt for confirmation")

	return cmd
}
