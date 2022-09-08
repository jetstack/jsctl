package command

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/jetstack/jsctl/internal/auth"
	"github.com/jetstack/jsctl/internal/config"
	"github.com/spf13/cobra"
	"github.com/toqueteos/webbrowser"
	"golang.org/x/oauth2"
)

// Auth returns a cobra.Command instance that is the root for all "jsctl auth" subcommands.
func Auth() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Subcommands for authentication",
	}

	cmd.AddCommand(
		authLogin(),
		authLogout(),
	)

	return cmd
}

func authLogin() *cobra.Command {
	var credentials string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Performs the authentication flow to allow access to other commands",
		Run: run(func(ctx context.Context, args []string) error {
			oAuthConfig := auth.GetOAuthConfig()

			var err error
			var token *oauth2.Token
			if credentials != "" {
				token, err = loginWithCredentials(ctx, oAuthConfig, credentials)
			} else {
				token, err = loginWithOAuth(ctx, oAuthConfig)
			}

			if err != nil {
				return fmt.Errorf("failed to obtain token: %w", err)
			}

			if err = auth.SaveOAuthToken(token); err != nil {
				return fmt.Errorf("failed to save token: %w", err)
			}

			fmt.Println("Login succeeded")

			err = config.Create(&config.Config{})
			switch {
			case errors.Is(err, config.ErrConfigExists):
				break
			case err != nil:
				return fmt.Errorf("failed to create configuration file: %w", err)
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
		"The location of a credentials file to use instead of the normal oauth login flow",
	)

	return cmd
}

func authLogout() *cobra.Command {
	return &cobra.Command{
		Use: "logout",
		Run: run(func(ctx context.Context, args []string) error {
			err := auth.DeleteOAuthToken()
			switch {
			case errors.Is(err, auth.ErrNoToken):
				return fmt.Errorf("host contains no authentication data")
			case err != nil:
				return fmt.Errorf("failed to remove authentication data: %w", err)
			default:
				fmt.Println("You were logged out successfully")
				return nil
			}
		}),
	}
}

func loginWithOAuth(ctx context.Context, oAuthConfig *oauth2.Config) (*oauth2.Token, error) {
	url, state := auth.GetOAuthURLAndState(oAuthConfig)

	if err := webbrowser.Open(url); err != nil {
		fmt.Printf("Navigate to the URL below to login:\n%s\n", url)
	} else {
		fmt.Println("You will be taken to your browser for authentication")
	}

	token, err := auth.WaitForOAuthToken(ctx, oAuthConfig, state)
	if err != nil {
		return nil, err
	}

	return token, nil
}

func loginWithCredentials(ctx context.Context, oAuthConfig *oauth2.Config, location string) (*oauth2.Token, error) {
	credentials, err := auth.LoadCredentials(location)
	switch {
	case errors.Is(err, auth.ErrNoCredentials):
		return nil, fmt.Errorf("no service account was found at: %s", location)
	case err != nil:
		return nil, fmt.Errorf("failed to read service account key: %w", err)
	}

	token, err := auth.GetOAuthTokenForCredentials(ctx, oAuthConfig, credentials)
	if err != nil {
		return nil, err
	}

	return token, nil
}
