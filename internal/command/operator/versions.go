package operator

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jetstack/jsctl/internal/command/types"
	"github.com/jetstack/jsctl/internal/operator"
)

func Versions(run types.RunFunc) *cobra.Command {
	return &cobra.Command{
		Use:   "versions",
		Short: "Outputs all available versions of the jetstack operator",
		Args:  cobra.ExactArgs(0),
		Run: run(func(ctx context.Context, args []string) error {
			versions, err := operator.Versions()
			if err != nil {
				return fmt.Errorf("failed to get operator versions: %w", err)
			}

			for _, version := range versions {
				fmt.Println(version)
			}

			return nil
		}),
	}
}
