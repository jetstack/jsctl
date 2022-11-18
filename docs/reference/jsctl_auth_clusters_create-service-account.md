## jsctl auth clusters create-service-account

Create a new Jetstack Secure service account for a cluster agent

### Synopsis

Generate a new service account for a Jetstack Secure cluster agent 
This is only needed if you are not deploying the agent with jsctl.
Output can be json formatted or as Kubernetes Secret.


```
jsctl auth clusters create-service-account name [flags]
```

### Options

```
      --format string             The desired output format, valid options: [jsonKeyData, secret] (default "jsonKeyData")
  -h, --help                      help for create-service-account
      --secret-name string        If using the 'secret' format, the name of the secret to create (default "agent-credentials")
      --secret-namespace string   If using the 'secret' format, the namespace of the secret to create (default "jetstack-secure")
```

### Options inherited from parent commands

```
      --api-url string      Base URL of the control-plane API (default "https://platform.jetstack.io")
      --config string       Location of the user's jsctl config directory (default "HOME or USERPROFILE/.jsctl")
      --kubeconfig string   Location of the user's kubeconfig file for applying directly to the cluster (default "~/.kube/config")
      --stdout              If provided, manifests are written to stdout rather than applied to the current cluster
```

### SEE ALSO

* [jsctl auth clusters](jsctl_auth_clusters.md)	 - 

