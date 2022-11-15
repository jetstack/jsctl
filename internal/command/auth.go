package command

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/golang-jwt/jwt/v4"
	"github.com/spf13/cobra"
	"github.com/toqueteos/webbrowser"
	"golang.org/x/oauth2"

	"github.com/jetstack/jsctl/internal/auth"
	"github.com/jetstack/jsctl/internal/config"
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
		authStatus(),
	)

	return cmd
}

func authStatus() *cobra.Command {
	var credentials string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Print the logged in account and token location",
		Args:  cobra.ExactArgs(0),
		Run: run(func(ctx context.Context, args []string) error {
			var token *oauth2.Token
			var err error
			var tokenPath string
			if credentials != "" {
				tokenPath = credentials
				token, err = loginWithCredentials(ctx, auth.GetOAuthConfig(), credentials)
				if err != nil {
					return fmt.Errorf("failed to login with credentials file %q: %w", credentials, err)
				}
			} else {
				tokenPath, err = auth.DetermineTokenFilePath(ctx)
				if err != nil {
					return fmt.Errorf("failed to determine token path: %w", err)
				}
				if _, err := os.Stat(tokenPath); errors.Is(err, os.ErrNotExist) {
					return fmt.Errorf("token missing at %s", tokenPath)
				}

				fmt.Println("Token path:", tokenPath)

				token, err = auth.LoadOAuthToken(ctx)
				if err != nil {
					fmt.Println("Not logged in")
					return nil
				}
			}

			claims := jwt.MapClaims{}
			_, err = jwt.ParseWithClaims(token.AccessToken, claims, func(token *jwt.Token) (interface{}, error) {
				return []byte(""), nil
			})

			email, ok := claims["https://jetstack.io/claims/name"].(string)
			if ok {
				fmt.Println("Logged in as:", email)
			}

			cnf, ok := config.FromContext(ctx)
			if !ok || cnf.Organization == "" {
				fmt.Println("You do not have an organization selected, select one using: \n\n\tjsctl config set organization [name]\n\n" +
					"To view organizations you have access to, list them using: \n\n\tjsctl organizations list")
				return nil
			}
			fmt.Println("Current Organization:", cnf.Organization)

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

func authLogin() *cobra.Command {
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

func authLogout() *cobra.Command {
	return &cobra.Command{
		Use:  "logout",
		Args: cobra.ExactArgs(0),
		Run: run(func(ctx context.Context, args []string) error {
			err := auth.DeleteOAuthToken(ctx)
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

func loginWithOAuth(ctx context.Context, oAuthConfig *oauth2.Config, disconnected bool) (*oauth2.Token, error) {
	url, state := auth.GetOAuthURLAndState(oAuthConfig)

	// disconnected can be set to true when the browser and terminal are not running
	// on the same machine.
	if disconnected {
		fmt.Printf("Navigate to the URL below to login:\n%s\n", url)
		token, err := auth.WaitForOAuthTokenCommandLine(ctx, oAuthConfig, state)
		if err != nil {
			return nil, fmt.Errorf("failed to obtain token: %w", err)
		}
		return token, nil
	}

	fmt.Println("Opening browser to:", url)

	if err := webbrowser.Open(url); err != nil {
		fmt.Printf("Navigate to the URL below to login:\n%s\n", url)
	} else {
		fmt.Println("You will be taken to your browser for authentication")
	}

	token, err := auth.WaitForOAuthTokenCallback(ctx, oAuthConfig, state)
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
