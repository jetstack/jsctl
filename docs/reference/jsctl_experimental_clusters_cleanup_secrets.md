## jsctl experimental clusters cleanup secrets

Perform operations to ensure secrets with issued X.509 certificates are not deleted when Jetstack Secure software is uninstalled

### Options

```
  -h, --help   help for secrets
```

### Options inherited from parent commands

```
      --api-url string      Base URL of the control-plane API (default "https://platform.jetstack.io")
      --config string       Location of the user's jsctl config directory (default "/Users/USER/.jsctl")
      --kubeconfig string   Location of the user's kubeconfig file for applying directly to the cluster (default "~/.kube/config")
      --stdout              If provided, manifests are written to stdout rather than applied to the current cluster
```

### SEE ALSO

* [jsctl experimental clusters cleanup](jsctl_experimental_clusters_cleanup.md)	 - Contains commands to prepare a cluster for the uninstallation of Jetstack Secure software
* [jsctl experimental clusters cleanup secrets remove-certificate-owner-refs](jsctl_experimental_clusters_cleanup_secrets_remove-certificate-owner-refs.md)	 - Remove certificate owner references from secret resources

