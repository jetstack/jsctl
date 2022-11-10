package command

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/toqueteos/webbrowser"
	"gopkg.in/yaml.v2"

	"github.com/jetstack/jsctl/internal/client"
	"github.com/jetstack/jsctl/internal/cluster"
	"github.com/jetstack/jsctl/internal/config"
	"github.com/jetstack/jsctl/internal/kubernetes"
	"github.com/jetstack/jsctl/internal/kubernetes/status"
	"github.com/jetstack/jsctl/internal/prompt"
	"github.com/jetstack/jsctl/internal/table"
)

// Clusters returns a cobra.Command instance that is the root for all "jsctl clusters" subcommands.
func Clusters() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "clusters",
		Aliases: []string{"cluster"},
		Short:   "Subcommands for cluster management",
	}

	cmd.AddCommand(
		clustersConnect(),
		clustersList(),
		clustersDelete(),
		clustersView(),
		clustersStatus(),
	)

	return cmd
}

func clustersConnect() *cobra.Command {
	const defaultRegistry = "quay.io/jetstack"
	var registry string

	cmd := &cobra.Command{
		Use:   "connect [name]",
		Short: "Creates a new cluster in the control plane and deploys the agent in your current kubenetes context",
		Args:  cobra.ExactValidArgs(1),
		Run: run(func(ctx context.Context, args []string) error {
			name := args[0]
			if name == "" {
				return errors.New("you must specify a cluster name")
			}

			cnf, ok := config.FromContext(ctx)
			if !ok || cnf.Organization == "" {
				return errNoOrganizationName
			}

			http := client.New(ctx, apiURL)

			serviceAccount, err := cluster.CreateServiceAccount(ctx, http, cnf.Organization, name)
			if err != nil {
				return fmt.Errorf("failed to create service account: %w", err)
			}

			var applier cluster.Applier
			if stdout {
				applier = kubernetes.NewStdOutApplier()
			} else {
				applier, err = kubernetes.NewKubeConfigApplier(kubeConfig)
				if err != nil {
					return err
				}
			}

			err = cluster.ApplyAgentYAML(ctx, applier, cluster.ApplyAgentYAMLOptions{
				Organization:   cnf.Organization,
				Name:           name,
				ServiceAccount: serviceAccount,
				ImageRegistry:  registry,
			})

			if err != nil {
				return fmt.Errorf("failed to generate agent YAML: %w", err)
			}

			fmt.Fprintf(os.Stderr, "Once connected, you can view the cluster in the dashboard:\n"+
				"https://platform.jetstack.io/org/%s/certinventory/cluster/%s\n", cnf.Organization, name)

			return nil
		}),
	}

	flags := cmd.PersistentFlags()
	flags.StringVar(&registry, "registry", defaultRegistry, "Specifies an alternative image registry to use for the agent image")

	return cmd
}

func clustersList() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Lists all clusters connected to the control plane for the organization",
		Args:  cobra.ExactValidArgs(0),
		Run: run(func(ctx context.Context, args []string) error {
			cnf, ok := config.FromContext(ctx)
			if !ok || cnf.Organization == "" {
				return errNoOrganizationName
			}

			http := client.New(ctx, apiURL)

			clusters, err := cluster.List(ctx, http, cnf.Organization)
			if err != nil {
				return fmt.Errorf("failed to list clusters: %w", err)
			}

			if jsonOut {
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent(" ", " ")
				return encoder.Encode(clusters)
			}

			tbl := table.NewBuilder([]string{
				"NAME",
				"LAST UPDATED",
			})

			for _, cl := range clusters {
				if cl.IsDemoData {
					continue
				}

				lastUpdated := "N/A"
				if cl.CertInventoryLastUpdated != nil {
					lastUpdated = cl.CertInventoryLastUpdated.Format("2006-01-02 15:04:05")
				}

				tbl.AddRow(cl.Name, lastUpdated)
			}

			return tbl.Build(os.Stdout)
		}),
	}

	flags := cmd.PersistentFlags()
	flags.BoolVar(&jsonOut, "json", false, "Output clusters in JSON format")

	return cmd
}

func clustersDelete() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete [name]",
		Short: "Deletes a cluster from the organization",
		Args:  cobra.ExactValidArgs(1),
		Run: run(func(ctx context.Context, args []string) error {
			cnf, ok := config.FromContext(ctx)
			if !ok || cnf.Organization == "" {
				return errNoOrganizationName
			}

			http := client.New(ctx, apiURL)
			name := args[0]
			if name == "" {
				return errors.New("you must specify a cluster name")
			}

			if !force {
				ok, err := prompt.YesNo(os.Stdin, os.Stdout, "Are you sure you want to delete cluster %s from organization %s?", name, cnf.Organization)
				switch {
				case err != nil:
					return fmt.Errorf("failed to prompt: %w", err)
				case !ok:
					return nil
				}
			}

			err := cluster.Delete(ctx, http, cnf.Organization, name)
			switch {
			case errors.Is(err, cluster.ErrNoCluster):
				return fmt.Errorf("cluster %s does not exist in organization %s", name, cnf.Organization)
			case err != nil:
				return fmt.Errorf("failed to delete cluster: %w", err)
			}

			fmt.Printf("Cluster %s was successfully deleted\n", name)
			return nil
		}),
	}

	flags := cmd.PersistentFlags()
	flags.BoolVar(&force, "force", false, "Do not prompt for confirmation")

	return cmd
}

func clustersView() *cobra.Command {
	return &cobra.Command{
		Use:   "view [name]",
		Short: "Opens a browser window to the cluster's dashboard",
		Args:  cobra.ExactValidArgs(1),
		Run: run(func(ctx context.Context, args []string) error {
			cnf, ok := config.FromContext(ctx)
			if !ok || cnf.Organization == "" {
				return errNoOrganizationName
			}

			http := client.New(ctx, apiURL)
			name := args[0]
			if name == "" {
				return errors.New("you must specify a cluster name")
			}

			clusters, err := cluster.List(ctx, http, cnf.Organization)
			if err != nil {
				return fmt.Errorf("failed to list clusters: %w", err)
			}

			const urlFormat = "https://platform.jetstack.io/org/%s/certinventory/cluster/%s/certificates"
			for _, cl := range clusters {
				if cl.Name == name {
					url := fmt.Sprintf(urlFormat, cnf.Organization, name)
					if err = webbrowser.Open(url); err != nil {
						fmt.Printf("Navigate to the URL below to view your cluster:\n%s\n", url)
					}

					return nil
				}
			}

			return fmt.Errorf("cluster %s does not exist in organization %s", name, cnf.Organization)
		}),
	}
}

func clustersStatus() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Prints information about the state in the currently configured cluster in kubeconfig",
		Long:  "The information printed by this command can be used to determine the state of a cluster prior to installing Jetstack Secure.",
		Args:  cobra.ExactValidArgs(0),
		Run: run(func(ctx context.Context, args []string) error {
			kubeCfg, err := kubernetes.NewConfig(kubeConfig)
			if err != nil {
				return err
			}

			s, err := status.GatherClusterStatus(ctx, kubeCfg)
			if err != nil {
				return fmt.Errorf("failed to gather cluster status: %w", err)
			}

			// marshal the status as yaml
			y, err := yaml.Marshal(s)
			if err != nil {
				return fmt.Errorf("failed to marshal status: %w", err)
			}

			fmt.Fprintf(os.Stdout, "%s", string(y))

			return nil
		}),
	}
}
