package command

import (
	"context"
	"errors"
	"fmt"

	"github.com/jetstack/jsctl/internal/client"
	"github.com/jetstack/jsctl/internal/config"
	"github.com/jetstack/jsctl/internal/organization"
	"github.com/spf13/cobra"
)

// Config returns a cobra.Command instance that is the root for all "jsctl config" subcommands.
func Config() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "configuration",
		Short:   "Subcommands for configuration management",
		Aliases: []string{"config", "conf"},
	}

	cmd.AddCommand(
		configSet(),
	)

	return cmd
}

func configSet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set a configuration value",
	}

	cmd.AddCommand(
		configSetOrganization(),
	)

	return cmd
}

func configSetOrganization() *cobra.Command {
	return &cobra.Command{
		Use:   "organization [value]",
		Short: "Set your current organization",
		Args:  cobra.ExactValidArgs(1),
		Run: run(func(ctx context.Context, args []string) error {
			name := args[0]
			if name == "" {
				return errors.New("you must specify an organization name")
			}

			http := client.New(ctx, apiURL)
			organizations, err := organization.List(ctx, http)
			if err != nil {
				return fmt.Errorf("failed to list organizations: %w", err)
			}

			found := false
			for _, org := range organizations {
				found = org.ID == name
				if found {
					break
				}
			}

			if !found {
				return fmt.Errorf("organization %s does not exist or you do not have access to it.\nTo see which "+
					"organizations you have access to, use:\n\n\tjsctl organizations list", name)
			}

			cnf, ok := config.FromContext(ctx)
			if !ok {
				return errors.New("failed to load configuration")
			}

			cnf.Organization = name
			if err = config.Save(cnf); err != nil {
				return fmt.Errorf("failed to save configuration: %w", err)
			}

			fmt.Printf("Your organization has been changed to %s\n", name)
			return nil
		}),
	}
}
