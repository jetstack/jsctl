package types

import (
	"context"

	"github.com/spf13/cobra"
)

// RunFunc is the function signature for the Run method of a cobra.Command.
type RunFunc func(func(ctx context.Context, args []string) error) func(cmd *cobra.Command, args []string)
