package command

import (
	"github.com/spf13/cobra"

	"github.com/jetstack/jsctl/internal/command/clusters"
)

// Clusters returns a cobra.Command instance that is the root for all "jsctl clusters" subcommands.
func Clusters() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "clusters",
		Aliases: []string{"cluster"},
		Short:   "Subcommands for cluster management",
	}

	cmd.AddCommand(
		clusters.Connect(run, kubeConfig, apiURL, useStdout),
		clusters.List(run, apiURL),
		clusters.Delete(run, apiURL),
		clusters.View(run, apiURL),
		clusters.Status(run, kubeConfig),
	)

	return cmd
}
