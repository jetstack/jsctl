## jsctl experimental clusters cleanup secrets remove-certificate-owner-refs

Remove certificate owner references from secret resources

### Synopsis

Removing Certificate owner references from secrets allows the uninstallation of cert-manager (including CRDs) without deleting the secrets that contain the issued X.509 certificates. This allows the uninstallation of cert-manager without causing application downtime or unneccessary certificate re-issuance. After cert-manager is re-installed and the Certificate resources are be re-applied, the existing secrets will be picked up for the Certificates.

```
jsctl experimental clusters cleanup secrets remove-certificate-owner-refs [flags]
```

### Options

```
  -h, --help   help for remove-certificate-owner-refs
```

### Options inherited from parent commands

```
      --api-url string      Base URL of the control-plane API (default "https://platform.jetstack.io")
      --config string       Location of the user's jsctl config directory (default "/Users/USER/.jsctl")
      --kubeconfig string   Location of the user's kubeconfig file for applying directly to the cluster (default "~/.kube/config")
      --stdout              If provided, manifests are written to stdout rather than applied to the current cluster
```

### SEE ALSO

* [jsctl experimental clusters cleanup secrets](jsctl_experimental_clusters_cleanup_secrets.md)	 - Perform operations to ensure secrets with issued X.509 certificates are not deleted when Jetstack Secure software is uninstalled

