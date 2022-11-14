package clusters

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/jetstack/jsctl/internal/command/types"
	"github.com/jetstack/jsctl/internal/kubernetes"
	"github.com/jetstack/jsctl/internal/kubernetes/status"
)

// Status returns a new command that shows the status of a cluster resources
func Status(run types.RunFunc, kubeConfigPath string) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Prints information about the state in the currently configured cluster in kubeconfig",
		Long:  "The information printed by this command can be used to determine the state of a cluster prior to installing Jetstack Secure.",
		Args:  cobra.ExactValidArgs(0),
		Run: run(func(ctx context.Context, args []string) error {
			kubeCfg, err := kubernetes.NewConfig(kubeConfigPath)
			if err != nil {
				return err
			}

			s, err := status.GatherClusterStatus(ctx, kubeCfg)
			if err != nil {
				return fmt.Errorf("failed to gather cluster status: %w", err)
			}

			// marshal the status as yaml as a simple means of display for now
			y, err := yaml.Marshal(s)
			if err != nil {
				return fmt.Errorf("failed to marshal status: %w", err)
			}

			fmt.Fprintf(os.Stdout, "%s", string(y))

			return nil
		}),
	}
}
