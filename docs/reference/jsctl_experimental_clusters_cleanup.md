## jsctl experimental clusters cleanup

Contains commands to prepare a cluster for the uninstallation of Jetstack Secure software

### Options

```
  -h, --help   help for cleanup
```

### Options inherited from parent commands

```
      --api-url string      Base URL of the control-plane API (default "https://platform.jetstack.io")
      --config string       Location of the user's jsctl config directory (default "HOME or USERPROFILE/.jsctl")
      --kubeconfig string   Location of the user's kubeconfig file for applying directly to the cluster (default "~/.kube/config")
      --stdout              If provided, manifests are written to stdout rather than applied to the current cluster
```

### SEE ALSO

* [jsctl experimental clusters](jsctl_experimental_clusters.md)	 - Experimental clusters commands
* [jsctl experimental clusters cleanup secrets](jsctl_experimental_clusters_cleanup_secrets.md)	 - Perform operations to ensure secrets with issued X.509 certificates are not deleted when Jetstack Secure software is uninstalled

