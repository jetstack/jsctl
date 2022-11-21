package command

import (
	"github.com/spf13/cobra"

	"github.com/jetstack/jsctl/internal/command/operator"
)

// Operator returns a cobra.Command instance that is the root for all "jsctl operator" subcommands.
func Operator() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "operator",
		Aliases: []string{"operators", "op"},
		Short:   "Subcommands for managing the Jetstack operator",
		Long: `
These commands cover the deployment of the operator and the
management of 'Installation' resources. Get started by deploying
the operator with "jsctl operator deploy --help"`,
	}

	cmd.AddCommand(
		operator.Deploy(run, &useStdout, &apiURL, &kubeConfig),
		operator.Versions(run),
		operatorInstallations(),
	)

	return cmd
}

func operatorInstallations() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "installations",
		Aliases: []string{"installation"},
		Short:   "Subcommands for managing operator installation resources",
	}

	cmd.AddCommand(
		operator.InstallationsApply(run, &useStdout, &apiURL, &kubeConfig),
		operator.InstallationStatus(run, &useStdout, &kubeConfig),
	)

	return cmd
}
