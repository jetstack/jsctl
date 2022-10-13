package command

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func Version(version *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "view the version, commit and build date of jsctl",
		Args:  cobra.ExactArgs(0),
		Run: run(func(ctx context.Context, args []string) error {
			fmt.Println(*version)
			if strings.Contains(*version, "dev") {
				fmt.Println("note: this is either a development build or the version was not injected at build-time")
			}
			return nil
		}),
	}

	return cmd
}
