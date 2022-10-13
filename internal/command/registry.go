package command

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jetstack/jsctl/internal/client"
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
			configDir, err := os.UserConfigDir()
			if err != nil {
				return err
			}

			fmt.Println("Checking for existing credentials in", configDir)

			jscpClient := client.New(ctx, apiURL)

			_, err = registry.FetchOrLoadJetstackSecureEnterpriseRegistryCredentials(ctx, jscpClient, configDir)
			if err != nil {
				return err
			}

			status, err := registry.StatusJetstackSecureEnterpriseRegistry(configDir)
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
			// TODO: it'd be nice to get this from the ctx config so that
			//  operations can be performed relative to the loaded config
			configDir, err := os.UserConfigDir()
			if err != nil {
				return err
			}

			fmt.Println("Checking for existing credentials in", configDir)

			status, err := registry.StatusJetstackSecureEnterpriseRegistry(configDir)
			if err != nil {
				return err
			}

			fmt.Println(status)

			return nil
		}),
	}
}
