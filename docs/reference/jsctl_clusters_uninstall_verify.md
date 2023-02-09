## jsctl clusters uninstall verify

Check that a cluster is ready to have Jetstack Software uninstalled

### Synopsis

Runs the following checks:
* Checks secrets containing certificates are safe from garbage collection
* Checks for any upcoming renewals
* Checks for certificates currently being issued
* Checks for stuck certificate requests


```
jsctl clusters uninstall verify [flags]
```

### Options

```
  -h, --help   help for verify
```

### Options inherited from parent commands

```
      --api-url string      Base URL of the control-plane API (default "https://platform.jetstack.io")
      --config string       Location of the user's jsctl config directory (default "HOME or USERPROFILE/.jsctl")
      --kubeconfig string   Location of the user's kubeconfig file for applying directly to the cluster (default "~/.kube/config")
      --stdout              If provided, manifests are written to stdout rather than applied to the current cluster
```

### SEE ALSO

* [jsctl clusters uninstall](jsctl_clusters_uninstall.md)	 - Contains commands to check a cluster before the uninstallation of Jetstack Secure software

