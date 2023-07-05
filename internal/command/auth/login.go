package auth

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/oauth2"

	"github.com/jetstack/jsctl/internal/auth"
	"github.com/jetstack/jsctl/internal/client"
	"github.com/jetstack/jsctl/internal/command/types"
	"github.com/jetstack/jsctl/internal/config"
	"github.com/jetstack/jsctl/internal/organization"
)

func Login(run types.RunFunc) *cobra.Command {
	var credentials string
	var disconnected bool
	var apiURL string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Performs the authentication flow to allow access to other commands",
		Args:  cobra.ExactArgs(0),
		Run: run(func(ctx context.Context, args []string) error {
			oAuthConfig := auth.GetOAuthConfig()

			var err error
			var token *oauth2.Token
			if credentials != "" {
				token, err = loginWithCredentials(ctx, oAuthConfig, credentials)
			} else {
				token, err = loginWithOAuth(ctx, oAuthConfig, disconnected)
			}

			if err != nil {
				return fmt.Errorf("failed to obtain token: %w", err)
			}

			if err = auth.SaveOAuthToken(ctx, token); err != nil {
				return fmt.Errorf("failed to save token: %w", err)
			}

			// update the context with the new token
			ctx = auth.TokenToContext(ctx, token)

			fmt.Println("Login succeeded")

			var cnf *config.Config

			// if the user already has an organization selected, we don't need to do anything
			cnf, ok := config.FromContext(ctx)
			if ok && cnf.Organization != "" {
				return nil
			}
			// initiate empty config
			if !ok {
				cnf = &config.Config{}
			}

			http := client.New(ctx, apiURL)
			organizations, err := organization.List(ctx, http)
			if err != nil {
				return fmt.Errorf("failed to list organizations: %w", err)
			}

			fmt.Println()

			if len(organizations) == 1 {
				cnf.Organization = organizations[0].ID

				fmt.Println("Automatically selected the only organization you have access to: " + organizations[0].ID)
			} else {
				fmt.Println(
					"You do not have an organization selected, select one using:\n" +
						"\n" +
						"\tjsctl config set organization [name]\n" +
						"\n" +
						"You have access to the following organizations (run 'jsctl organizations list'):",
				)

				for _, org := range organizations {
					fmt.Println("  - " + org.ID)
				}
			}

			// save the configuration to disk
			if err = config.Save(ctx, cnf); err != nil {
				return fmt.Errorf("failed to save configuration: %w", err)
			}

			return nil
		}),
	}

	flags := cmd.PersistentFlags()
	flags.StringVar(
		&credentials,
		"credentials",
		os.Getenv("JSCTL_CREDENTIALS"),
		"The location of service account credentials file to use instead of the normal oauth login flow",
	)
	flags.BoolVar(
		&disconnected,
		"disconnected",
		false,
		"Use a disconnected login flow where browser and terminal are not running on the same machine",
	)
	flags.StringVar(&apiURL, "api-url", "https://platform.jetstack.io", "Base URL of the control-plane API")

	return cmd
}
