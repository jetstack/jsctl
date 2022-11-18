package auth

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/oauth2"

	"github.com/jetstack/jsctl/internal/auth"
	"github.com/jetstack/jsctl/internal/command/types"
	"github.com/jetstack/jsctl/internal/config"
)

func Login(run types.RunFunc) *cobra.Command {
	var credentials string
	var disconnected bool

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

			fmt.Println("Login succeeded")

			err = config.Save(ctx, &config.Config{})
			if err != nil {
				return fmt.Errorf("failed to save configuration: %w", err)
			}

			cnf, ok := config.FromContext(ctx)
			if !ok || cnf.Organization == "" {
				fmt.Println("You do not have an organization selected, select one using: \n\n\tjsctl config set organization [name]\n\n" +
					"To view organizations you have access to, list them using: \n\n\tjsctl organizations list")
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

	return cmd
}
