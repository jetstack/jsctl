package command

import (
	"github.com/spf13/cobra"

	"github.com/jetstack/jsctl/internal/command/clusters"
)

// Experimental returns a cobra.Command instance that is the root for all
// "jsctl experimental" subcommands.
func Experimental() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "experimental",
		Short: "Experimental jsctl commands",
	}

	experimentalClustersCommands := &cobra.Command{
		Use:   "clusters ",
		Short: "Experimental clusters commands",
	}

	experimentalClustersCommands.AddCommand(
		clusters.CleanUp(run, &kubeConfig),
		clusters.Backup(run, &kubeConfig),
	)

	cmd.AddCommand(experimentalClustersCommands)

	return cmd
}
