package clusters

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jetstack/jsctl/internal/command/types"
	"github.com/jetstack/jsctl/internal/kubernetes"
	"github.com/jetstack/jsctl/internal/kubernetes/backup"
	"github.com/jetstack/jsctl/internal/kubernetes/clients"
)

func Backup(run types.RunFunc, kubeConfigPath *string) *cobra.Command {
	var formatResources bool
	var outputFormat string

	var includeCertificates bool
	var includeIssuers bool
	var includeCertificateRequestPolicies bool

	cmd := &cobra.Command{
		Use:   "backup",
		Short: "This command outputs the YAML data of Jetstack Secure relevant resources in the cluster",
		Args:  cobra.MatchAll(cobra.ExactArgs(0)),
		Run: run(func(ctx context.Context, args []string) error {
			kubeCfg, err := kubernetes.NewConfig(*kubeConfigPath)
			if err != nil {
				return err
			}

			opts := backup.ClusterBackupOptions{
				RestConfig: kubeCfg,

				FormatResources: formatResources,

				IncludeCertificates:               includeCertificates,
				IncludeIssuers:                    includeIssuers,
				IncludeCertificateRequestPolicies: includeCertificateRequestPolicies,
			}

			clusterBackup, err := backup.FetchClusterBackup(context.Background(), opts)
			if err != nil {
				return fmt.Errorf("error backing up cluster: %s", err)
			}

			backupYAML, err := clusterBackup.ToYAML()
			if err != nil {
				return fmt.Errorf("error converting backup to YAML: %s", err)
			}

			fmt.Fprintf(os.Stdout, "%s", backupYAML)

			return nil
		}),
	}

	allIssuers, err := clients.ListSupportedIssuers()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error determining supported issuers, this is a bug: %s", err)
		os.Exit(1)
	}
	allIssuersString := allIssuers.String()

	fmt.Println(allIssuersString)

	flags := cmd.PersistentFlags()
	flags.BoolVar(&formatResources, "format-resources", true, "if set, will remove some fields from resources such as status and metadata to allow them to be cleanly applied later")
	flags.StringVar(&outputFormat, "format", "yaml", "output format, one of: yaml, json")

	flags.BoolVar(&includeCertificates, "include-certificates", true, "if set, certificate resources will be included in the backup. Note: ingress-shim managed certificates are not included since they are automatically generated.")
	flags.BoolVar(&includeIssuers, "include-issuers", true, fmt.Sprintf("if set, issuer resources will be included in the backup (supports: %s)", allIssuersString))
	flags.BoolVar(&includeCertificateRequestPolicies, "include-certificate-request-policies", true, "if set, certificate request policy resources will be included in the backup")

	return cmd
}
