package command

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/jetstack/jsctl/internal/auth"
	"github.com/jetstack/jsctl/internal/client"
	"github.com/jetstack/jsctl/internal/config"
	"github.com/jetstack/jsctl/internal/organization"
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
		configShow(),
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

func configShow() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "View your current configuration values",
		Args:  cobra.MatchAll(cobra.ExactArgs(0)),
		Run: run(func(ctx context.Context, args []string) error {
			cnf, ok := config.FromContext(ctx)
			if !ok {
				return errors.New("config was not present, have you logged in? try jsctl auth login")
			}

			configDir, ok := ctx.Value(config.ContextKey{}).(string)
			if !ok {
				configDir = "unknown"
			}

			fmt.Fprintln(os.Stderr, "Configuration loaded from", configDir)

			yamlBytes, err := yaml.Marshal(cnf)
			if err != nil {
				return fmt.Errorf("failed to marshal configuration: %w", err)
			}

			fmt.Fprintf(os.Stderr, string(yamlBytes))

			return nil
		}),
	}
}

func configSetOrganization() *cobra.Command {
	return &cobra.Command{
		Use:   "organization name",
		Short: "Set your current organization",
		Args:  cobra.MatchAll(cobra.ExactArgs(1)),
		Run: run(func(ctx context.Context, args []string) error {
			name := args[0]
			if name == "" {
				return errors.New("you must specify an organization name")
			}

			// users must be logged in to run this command. Organizations can
			// only be selected from the current token's organizations.
			_, ok := auth.TokenFromContext(ctx)
			if !ok {
				return fmt.Errorf("you must be logged in to run this command, run jsctl auth login")
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
			if err = config.Save(ctx, cnf); err != nil {
				return fmt.Errorf("failed to save configuration: %w", err)
			}

			fmt.Printf("Your organization has been changed to %s\n", name)
			return nil
		}),
	}
}
