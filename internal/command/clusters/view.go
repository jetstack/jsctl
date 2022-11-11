package clusters

import (
	"context"
	errors2 "errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/toqueteos/webbrowser"

	"github.com/jetstack/jsctl/internal/client"
	"github.com/jetstack/jsctl/internal/cluster"
	"github.com/jetstack/jsctl/internal/command/errors"
	"github.com/jetstack/jsctl/internal/command/types"
	"github.com/jetstack/jsctl/internal/config"
)

// View returns a new cobra.Command for viewing a cluster in the JSCP api
func View(run types.RunFunc, apiURL string) *cobra.Command {
	return &cobra.Command{
		Use:   "view [name]",
		Short: "Opens a browser window to the cluster's dashboard",
		Args:  cobra.ExactValidArgs(1),
		Run: run(func(ctx context.Context, args []string) error {
			cnf, ok := config.FromContext(ctx)
			if !ok || cnf.Organization == "" {
				return errors.ErrNoOrganizationName
			}

			http := client.New(ctx, apiURL)
			name := args[0]
			if name == "" {
				return errors2.New("you must specify a cluster name")
			}

			clusters, err := cluster.List(ctx, http, cnf.Organization)
			if err != nil {
				return fmt.Errorf("failed to list clusters: %w", err)
			}

			const urlFormat = "https://platform.jetstack.io/org/%s/certinventory/cluster/%s/certificates"
			for _, cl := range clusters {
				if cl.Name == name {
					url := fmt.Sprintf(urlFormat, cnf.Organization, name)
					if err = webbrowser.Open(url); err != nil {
						fmt.Printf("Navigate to the URL below to view your cluster:\n%s\n", url)
					}

					return nil
				}
			}

			return fmt.Errorf("cluster %s does not exist in organization %s", name, cnf.Organization)
		}),
	}
}
