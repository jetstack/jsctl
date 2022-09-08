package command

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/jetstack/jsctl/internal/client"
	"github.com/jetstack/jsctl/internal/organization"
	"github.com/jetstack/jsctl/internal/table"
	"github.com/spf13/cobra"
)

// Organizations returns a cobra.Command instance that is the root for all "jsctl organizations" subcommands.
func Organizations() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "organizations",
		Short:   "Subcommands for organization management",
		Aliases: []string{"organization", "org"},
	}

	cmd.AddCommand(
		organizationsList(),
	)

	return cmd
}

func organizationsList() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Lists all organizations the user has access to",
		Run: run(func(ctx context.Context, args []string) error {
			http := client.New(ctx, apiURL)

			organizations, err := organization.List(ctx, http)
			if err != nil {
				return fmt.Errorf("failed to list organizations: %w", err)
			}

			if jsonOut {
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent(" ", " ")
				return encoder.Encode(organizations)
			}

			tbl := table.NewBuilder([]string{
				"NAME",
				"ROLES",
			})

			for _, org := range organizations {
				tbl.AddRow(org.ID, strings.Join(org.Roles, ", "))
			}

			return tbl.Build(os.Stdout)
		}),
	}

	flags := cmd.PersistentFlags()
	flags.BoolVar(&jsonOut, "json", false, "Output organizations in JSON format")

	return cmd
}
