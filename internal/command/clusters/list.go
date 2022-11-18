package clusters

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jetstack/jsctl/internal/client"
	"github.com/jetstack/jsctl/internal/cluster"
	"github.com/jetstack/jsctl/internal/command/errors"
	"github.com/jetstack/jsctl/internal/command/types"
	"github.com/jetstack/jsctl/internal/config"
	"github.com/jetstack/jsctl/internal/table"
)

// List returns a new cobra.Command for listing clusters in the JSCP api
func List(run types.RunFunc, apiURL *string) *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Lists all clusters connected to the control plane for the organization",
		Args:  cobra.ExactValidArgs(0),
		Run: run(func(ctx context.Context, args []string) error {
			cnf, ok := config.FromContext(ctx)
			if !ok || cnf.Organization == "" {
				return errors.ErrNoOrganizationName
			}

			http := client.New(ctx, *apiURL)

			clusters, err := cluster.List(ctx, http, cnf.Organization)
			if err != nil {
				return fmt.Errorf("failed to list clusters: %w", err)
			}

			if jsonOut {
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent(" ", " ")
				return encoder.Encode(clusters)
			}

			tbl := table.NewBuilder([]string{
				"NAME",
				"LAST UPDATED",
			})

			for _, cl := range clusters {
				if cl.IsDemoData {
					continue
				}

				lastUpdated := "N/A"
				if cl.CertInventoryLastUpdated != nil {
					lastUpdated = cl.CertInventoryLastUpdated.Format("2006-01-02 15:04:05")
				}

				tbl.AddRow(cl.Name, lastUpdated)
			}

			return tbl.Build(os.Stdout)
		}),
	}

	flags := cmd.PersistentFlags()
	flags.BoolVar(&jsonOut, "json", false, "Output clusters in JSON format")

	return cmd
}
