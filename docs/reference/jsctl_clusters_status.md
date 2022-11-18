## jsctl clusters status

Prints information about the state in the currently configured cluster in kubeconfig

### Synopsis

The information printed by this command can be used to determine the state of a cluster prior to installing Jetstack Secure.

```
jsctl clusters status [flags]
```

### Options

```
  -h, --help   help for status
```

### Options inherited from parent commands

```
      --api-url string      Base URL of the control-plane API (default "https://platform.jetstack.io")
      --config string       Location of the user's jsctl config directory (default "HOME or USERPROFILE/.jsctl")
      --kubeconfig string   Location of the user's kubeconfig file for applying directly to the cluster (default "~/.kube/config")
      --stdout              If provided, manifests are written to stdout rather than applied to the current cluster
```

### SEE ALSO

* [jsctl clusters](jsctl_clusters.md)	 - Subcommands for cluster management

