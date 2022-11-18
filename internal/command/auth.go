package command

import (
	"github.com/spf13/cobra"

	"github.com/jetstack/jsctl/internal/command/auth"
)

// Auth returns a cobra.Command instance that is the root for all "jsctl auth" subcommands.
func Auth() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Subcommands for authentication",
	}

	cmd.AddCommand(
		auth.Login(run),
		auth.Logout(run),
		auth.Status(run),
	)

	return cmd
}
