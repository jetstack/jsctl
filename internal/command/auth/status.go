package auth

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/golang-jwt/jwt/v4"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"

	"github.com/jetstack/jsctl/internal/auth"
	"github.com/jetstack/jsctl/internal/command/types"
	"github.com/jetstack/jsctl/internal/config"
)

func Status(run types.RunFunc) *cobra.Command {
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
