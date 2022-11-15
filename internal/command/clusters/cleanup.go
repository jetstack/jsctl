package clusters

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/strategicpatch"

	"github.com/jetstack/jsctl/internal/command/types"
	"github.com/jetstack/jsctl/internal/kubernetes"
	"github.com/jetstack/jsctl/internal/kubernetes/clients"
	"github.com/jetstack/jsctl/internal/kubernetes/status/components"
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
			kubeCfg, err := kubernetes.NewConfig(kubeConfigPath)
			if err != nil {
				return err
			}

			podClient, err := clients.NewGenericClient[*corev1.Pod, *corev1.PodList](
				&clients.GenericClientOptions{
					RestConfig: kubeCfg,
					APIPath:    "/api/",
					Group:      corev1.GroupName,
					Version:    corev1.SchemeGroupVersion.Version,
					Kind:       "pods",
				},
			)

			var pods corev1.PodList
			err = podClient.List(ctx, &clients.GenericRequestOptions{}, &pods)

			md := components.MatchData{Pods: pods.Items}

			var certManagerStatus components.CertManagerStatus
			found, err := certManagerStatus.Match(&md)

			if !found {
				fmt.Fprintf(os.Stderr, "cert-manager not found, nothing to do\n")
				return nil
			}

			enableCertificateOwnerRefFlag := "enable-certificate-owner-ref"
			if found, _ := certManagerStatus.GetControllerFlagValue(enableCertificateOwnerRefFlag); found {
				fmt.Fprintf(os.Stderr, "cert-manager controller has --%s flag set, this must be removed from the controller's deployment pod spec before continuing.\n\n", enableCertificateOwnerRefFlag)
				fmt.Fprintf(os.Stderr, "If left set, the controller will re-add owner references to secrets, which will cause them to be deleted as part of the backup process when Certificates are removed from the cluster state. This will lead to reissueance of certificates on restore and could cause potential outages caused by CA rate limiting.\n\n")
				fmt.Fprintf(os.Stderr, "No cleanup action has been taken at this time\n")
				return nil
			}

			fmt.Fprintf(os.Stderr, "Checking for ownerReferences on secrets...\n")

			// if the flag is not found, we still want to check that the owner
			// references are not present. It can take some time for
			// cert-manager to remove them, even if cert-manager is running.

			secretsClient, err := clients.NewGenericClient[*corev1.Secret, *corev1.SecretList](
				&clients.GenericClientOptions{
					RestConfig: kubeCfg,
					APIPath:    "/api/",
					Group:      corev1.GroupName,
					Version:    corev1.SchemeGroupVersion.Version,
					Kind:       "secrets",
				},
			)

			var secretsList corev1.SecretList
			err = secretsClient.List(ctx, &clients.GenericRequestOptions{}, &secretsList)

			for _, secret := range secretsList.Items {
				hasCertificatOwnerRef := false
				for _, ownerRef := range secret.OwnerReferences {
					if ownerRef.Kind == "Certificate" {
						hasCertificatOwnerRef = true
						break
					}
				}

				if hasCertificatOwnerRef {
					fmt.Fprintf(os.Stderr, "Removing owner reference from %s/%s\n", secret.Namespace, secret.Name)
					newSecret := secret.DeepCopy()
					newSecret.OwnerReferences = []metav1.OwnerReference{}

					for _, ownerRef := range newSecret.OwnerReferences {
						if ownerRef.Kind != "Certificate" {
							newSecret.OwnerReferences = append(newSecret.OwnerReferences, ownerRef)
							break
						}
					}

					secretData, err := json.Marshal(secret)
					if err != nil {
						return fmt.Errorf("error marshalling secret: %s", err)
					}
					newSecretData, err := json.Marshal(newSecret)
					if err != nil {
						return fmt.Errorf("error marshalling new secret: %s", err)
					}

					patch, err := strategicpatch.CreateTwoWayMergePatch(secretData, newSecretData, corev1.Secret{})
					if err != nil {
						return fmt.Errorf("error creating patch for secret %s: %s", secret.Name, err)
					}

					err = secretsClient.Patch(ctx, &clients.GenericRequestOptions{Name: secret.Name, Namespace: secret.Namespace}, patch)
					if err != nil {
						return fmt.Errorf("error patching secret %s: %s", secret.Name, err)
					}
				}
			}

			return nil
		}),
	}
}
