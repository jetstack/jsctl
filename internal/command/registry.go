package command

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jetstack/jsctl/internal/auth"
	"github.com/jetstack/jsctl/internal/client"
	"github.com/jetstack/jsctl/internal/config"
	"github.com/jetstack/jsctl/internal/registry"
)

func Registry() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "registry",
		Short: "Subcommands for Jetstack Secure registry management",
	}

	cmd.AddCommand(registryAuth())

	return cmd
}

func registryAuth() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Subcommands for registry authentication",
	}

	cmd.AddCommand(registryAuthStatus())
	cmd.AddCommand(registryAuthInit())

	return cmd
}

func registryAuthInit() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Fetch or check the local registry credentials for the Jetstack Secure Enterprise registry",
		Args:  cobra.ExactArgs(0),
		Run: run(func(ctx context.Context, args []string) error {
			var err error

			// users must be logged in to run this command
			_, ok := auth.TokenFromContext(ctx)
			if !ok {
				return fmt.Errorf("you must be logged in to run this command, run jsctl auth login")
			}

			fmt.Println("Checking for existing credentials in", configDir)

			jscpClient := client.New(ctx, apiURL)

			_, err = registry.FetchOrLoadJetstackSecureEnterpriseRegistryCredentials(ctx, jscpClient)
			if err != nil {
				return err
			}

			status, err := registry.StatusJetstackSecureEnterpriseRegistry(ctx)
			if err != nil {
				return err
			}

			fmt.Println(status)

			return nil
		}),
	}
}

func registryAuthStatus() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Print the status of the local registry credentials",
		Args:  cobra.ExactArgs(0),
		Run: run(func(ctx context.Context, args []string) error {
			configDir, ok := ctx.Value(config.ContextKey{}).(string)
			if !ok {
				return fmt.Errorf("no config path provided")
			}

			fmt.Printf("Checking for existing credentials at path: %s\n", configDir)

			status, err := registry.StatusJetstackSecureEnterpriseRegistry(ctx)
			if err != nil {
				return err
			}

			fmt.Println(status)

			path, err := registry.PathJetstackSecureEnterpriseRegistry(ctx)
			if err != nil {
				return fmt.Errorf("failed to get path to registry credentials: %s", err)
			}

			fmt.Fprintf(os.Stderr, "Path to registry credentials: %s\n", path)

			return nil
		}),
	}
}
