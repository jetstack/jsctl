## jsctl clusters connect

Creates a new cluster in the control plane and deploys the agent in your current kubenetes context

```
jsctl clusters connect [name] [flags]
```

### Options

```
  -h, --help              help for connect
      --registry string   Specifies an alternative image registry to use for the agent image (default "quay.io/jetstack")
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

