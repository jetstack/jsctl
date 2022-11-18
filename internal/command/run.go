package command

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jetstack/jsctl/internal/auth"
	"github.com/jetstack/jsctl/internal/config"
)

// run is the wrapper function that is used to wrap all subcommands
func run(fn func(ctx context.Context, args []string) error) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		var err error
		ctx := cmd.Context()

		defaultConfigDir, err := config.DefaultConfigDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to determine default user config directory, using current directory\n")
			defaultConfigDir = "."
		}

		// if the user is using configDir defaulting, then we need to check for
		// legacy config directories and migrate them if they exist
		if configDir == defaultConfigDir {
			err := config.MigrateDefaultConfig(configDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to migrate legacy config directory: %s\n", err)
				os.Exit(1)
			}
		}

		// write the configuration directory to the context so that it can be
		// used in subcommands
		ctx = context.WithValue(ctx, config.ContextKey{}, configDir)

		// ensure that the config dir specified exists, this allows other
		// commands to write to sub paths of this directory without concern
		// for the config dir existing
		err = os.MkdirAll(configDir, 0700)
		if err != nil {
			exitf("failed to create config directory: %s", err)
		}

		token, err := auth.LoadOAuthToken(ctx)
		switch {
		case errors.Is(err, auth.ErrNoToken):
			break
		case err != nil:
			exitf("failed to load oauth token: %s", err)
		default:
			ctx = auth.TokenToContext(ctx, token)
		}

		cnf, err := config.Load(ctx)
		switch {
		case errors.Is(err, config.ErrNoConfiguration):
			break
		case err != nil:
			exitf("failed to load configuration: %s", err)
		default:
			ctx = config.ToContext(ctx, cnf)
		}

		if err = fn(ctx, args); err != nil {
			exitf(err.Error())
		}
	}
}

func exitf(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
