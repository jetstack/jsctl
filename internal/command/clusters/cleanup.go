package clusters

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jetstack/jsctl/internal/command/types"
)

// CleanUp returns a new command that wraps cluster clean up commands
func CleanUp(run types.RunFunc, kubeConfigPath string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Perform cleanup operations on a cluster's Kubernetes state",
	}

	cmd.AddCommand(secrets(run, kubeConfigPath))

	return cmd
}

func secrets(run types.RunFunc, kubeConfigPath string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secrets",
		Short: "Perform cleanup operations related to Kubernetes secrets",
	}

	cmd.AddCommand(removeSecretOwnerReferences(run, kubeConfigPath))

	return cmd
}

func removeSecretOwnerReferences(run types.RunFunc, kubeConfigPath string) *cobra.Command {
	return &cobra.Command{
		Use:   "remove-secret-owner-refs",
		Short: "Remove owner references to cert-manager resources from all secrets in the cluster",
		Long:  "Removing owner references from secrets allows cert-manager CRDs to be removed and the underlying secret data to be retained.",
		Args:  cobra.MatchAll(cobra.ExactArgs(0)),
		Run: run(func(ctx context.Context, args []string) error {
			fmt.Println("TODO: implement remove-secret-owner-refs")
			return nil
		}),
	}
}
