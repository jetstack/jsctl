# `jsctl cluster status` command

The cluster status command shows the status resource definitions and resources
in the cluster. This command exists to aid in the installation and maintenance
of Jetstack Secure.

## A component is not being correctly identified

If an item in this part of the output:

```
components:
...
```

appears to be incorrect, then you will need to alter the component's matching
code. These are found in the `internal/kubernetes/status/components/` directory.

Updating the `Match` implementation will allow the component to be correctly
identified, note, more information migth need to be supplied to the function
to robustly identify the component. This will require updates to the 
`installedComponent` interface.
