## jsctl clusters

Subcommands for cluster management

### Options

```
  -h, --help   help for clusters
```

### Options inherited from parent commands

```
      --api-url string      Base URL of the control-plane API (default "https://platform.jetstack.io")
      --config string       Location of the user's jsctl config directory (default "HOME or USERPROFILE/.jsctl")
      --kubeconfig string   Location of the user's kubeconfig file for applying directly to the cluster (default "~/.kube/config")
      --stdout              If provided, manifests are written to stdout rather than applied to the current cluster
```

### SEE ALSO

* [jsctl](jsctl.md)	 - Command-line tool for the Jetstack Secure Control Plane
* [jsctl clusters connect](jsctl_clusters_connect.md)	 - Creates a new cluster in the control plane and deploys the agent in your current kubenetes context
* [jsctl clusters delete](jsctl_clusters_delete.md)	 - Deletes a cluster from the organization
* [jsctl clusters list](jsctl_clusters_list.md)	 - Lists all clusters connected to the control plane for the organization
* [jsctl clusters status](jsctl_clusters_status.md)	 - Prints information about the state in the currently configured cluster in kubeconfig
* [jsctl clusters view](jsctl_clusters_view.md)	 - Opens a browser window to the cluster's dashboard

