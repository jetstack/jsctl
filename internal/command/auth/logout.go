package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jetstack/jsctl/internal/auth"
	"github.com/jetstack/jsctl/internal/command/types"
)

func Logout(run types.RunFunc) *cobra.Command {
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
