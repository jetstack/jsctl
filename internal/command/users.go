package command

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jetstack/jsctl/internal/client"
	errors2 "github.com/jetstack/jsctl/internal/command/errors"
	"github.com/jetstack/jsctl/internal/config"
	"github.com/jetstack/jsctl/internal/prompt"
	"github.com/jetstack/jsctl/internal/table"
	"github.com/jetstack/jsctl/internal/user"
)

// Users returns a cobra.Command instance that is the root for all "jsctl users" subcommands.
func Users() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "users",
		Aliases: []string{"user"},
		Short:   "Subcommands for user management",
	}

	cmd.AddCommand(
		usersList(),
		usersAdd(),
		usersRemove(),
	)

	return cmd
}

func usersList() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Lists all users the within the current organization",
		Args:  cobra.ExactArgs(0),
		Run: run(func(ctx context.Context, args []string) error {
			http := client.New(ctx, apiURL)
			cnf, ok := config.FromContext(ctx)
			if !ok || cnf.Organization == "" {
				return errors2.ErrNoOrganizationName
			}

			users, err := user.List(ctx, http, cnf.Organization)
			if err != nil {
				return fmt.Errorf("failed to list users: %w", err)
			}

			if jsonOut {
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent(" ", " ")
				return encoder.Encode(users)
			}

			tbl := table.NewBuilder([]string{
				"EMAIL",
				"ROLES",
			})

			for _, u := range users {
				tbl.AddRow(u.Email, strings.Join(u.Roles, ", "))
			}

			return tbl.Build(os.Stdout)
		}),
	}

	flags := cmd.PersistentFlags()
	flags.BoolVar(&jsonOut, "json", false, "Output users in JSON format")

	return cmd
}

func usersAdd() *cobra.Command {
	var admin bool

	cmd := &cobra.Command{
		Use:   "add email",
		Short: "Add a user to the current organization",
		Args:  cobra.MatchAll(cobra.ExactArgs(1)),
		Run: run(func(ctx context.Context, args []string) error {
			http := client.New(ctx, apiURL)
			cnf, ok := config.FromContext(ctx)
			if !ok || cnf.Organization == "" {
				return errors2.ErrNoOrganizationName
			}

			email := args[0]
			if email == "" {
				return errors.New("you must specify an email address")
			}

			if _, err := user.Add(ctx, http, cnf.Organization, email, admin); err != nil {
				return fmt.Errorf("failed to add user %s to organization: %w", email, err)
			}

			fmt.Printf("User %s was successfully added to organization %s\n", email, cnf.Organization)
			return nil
		}),
	}

	flags := cmd.PersistentFlags()
	flags.BoolVar(&admin, "admin", false, "Add the user as an organization administrator")

	return cmd
}

func usersRemove() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "remove email",
		Short: "Remove a user from the current organization",
		Args:  cobra.MatchAll(cobra.ExactArgs(1)),
		Run: run(func(ctx context.Context, args []string) error {
			http := client.New(ctx, apiURL)
			cnf, ok := config.FromContext(ctx)
			if !ok || cnf.Organization == "" {
				return errors2.ErrNoOrganizationName
			}

			email := args[0]
			if email == "" {
				return errors.New("you must specify an email address")
			}

			if !force {
				ok, err := prompt.YesNo(os.Stdin, os.Stdout, "Are you sure you want to remove user %s from organization %s?", email, cnf.Organization)
				switch {
				case err != nil:
					return fmt.Errorf("failed to prompt: %w", err)
				case !ok:
					return nil
				}
			}

			err := user.Remove(ctx, http, cnf.Organization, email)
			switch {
			case errors.Is(err, user.ErrNoUser):
				return fmt.Errorf("user %s does not exist in organization %s", email, cnf.Organization)
			case err != nil:
				return fmt.Errorf("failed to remove user: %w", err)
			default:
				fmt.Printf("User %s was removed from organization %s\n", email, cnf.Organization)
				return nil
			}
		}),
	}

	flags := cmd.PersistentFlags()
	flags.BoolVar(&force, "force", false, "Do not prompt for confirmation")

	return cmd
}
