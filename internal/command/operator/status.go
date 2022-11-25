package operator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jetstack/jsctl/internal/command/types"
	"github.com/jetstack/jsctl/internal/kubernetes"
	"github.com/jetstack/jsctl/internal/kubernetes/clients"
	"github.com/jetstack/jsctl/internal/table"
)

func InstallationStatus(run types.RunFunc, useStdout *bool, kubeConfig *string) *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Output the status of all operator components",
		Args:  cobra.ExactArgs(0),
		Run: run(func(ctx context.Context, args []string) error {
			if *useStdout {
				return fmt.Errorf("cannot use --stdout flag with status command. When using --stdout, jsctl does not connect to kubernetes")
			}

			kubeCfg, err := kubernetes.NewConfig(*kubeConfig)
			if err != nil {
				return err
			}

			installationClient, err := clients.NewInstallationClient(kubeCfg)
			if err != nil {
				return err
			}

			// first check if the operator and CRD is installed, this allows a better error message to be shown
			crdClient, err := clients.NewCRDClient(kubeCfg)
			if err != nil {
				return err
			}
			present, err := crdClient.Present(ctx, &clients.GenericRequestOptions{Name: "installations.operator.jetstack.io"})
			if err != nil {
				return fmt.Errorf("failed to query installation CRDs: %w", err)
			}
			if !present {
				return fmt.Errorf("no installations.operator.jetstack.io CRD found in cluster %q, have you run 'jsctl operator deploy'?", kubeCfg.Host)
			}

			// next, get the status of the installation components
			statuses, err := installationClient.Status(ctx)
			switch {
			case errors.Is(err, clients.ErrNoInstallation):
				return fmt.Errorf("no installations.operator.jetstack.io resources found in cluster %q, have you run 'jsctl operator installations apply'?", kubeCfg.Host)
			case err != nil:
				return fmt.Errorf("failed to query installation: %w", err)
			}

			if jsonOut {
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent(" ", " ")
				return encoder.Encode(statuses)
			}

			tbl := table.NewBuilder([]string{
				"COMPONENT",
				"READY",
				"MESSAGE",
			})

			for _, status := range statuses {
				tbl.AddRow(status.Name, status.Ready, status.Message)
			}

			return tbl.Build(os.Stdout)
		}),
	}

	flags := cmd.PersistentFlags()
	flags.BoolVar(&jsonOut, "json", false, "Output statuses in JSON format")

	return cmd
}
