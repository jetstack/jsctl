# jsctl

**Note: currently this code can only be built and tested by those with access
to Jetstack private repos. The process for community involvement has yet to be
determined.**

jsctl is the command-line tool for interacting with the [Jetstack Secure Control Plane](https://platform.jetstack.io).

## Getting Started

Obtain a binary for your os and architecture from the [releases page](https://github.com/jetstack/jsctl/releases) and
place it somewhere within your `PATH` environment variable.

Some commands make modifications to the Kubernetes cluster specified as your current context within your kubeconfig
file. Ensure you're set up to use the correct cluster using the `kubectl config use-context` command. By default,
the kubeconfig is expected at `~/.kube/config` but it can be set via the `KUBECONFIG` environment variable or by
providing the path via the `--kubeconfig` flag for commands that interact with clusters.

### Authentication

To authenticate, use the `jsctl auth login` command. It will open your default browser and navigate to the login screen.

In a non-interactive environment, or if the browser cannot be opened, it will print out a URL for you to visit:

```shell
jsctl auth login
> Navigate to the URL below to login:
> https://auth.jetstack.io/authorize
```

Once you have logged in, you should see a `Login Succeeded` message in your terminal. Check the browser window for any
errors.

To remove all authentication data from the host system, use the `jsctl auth logout` command.

#### Unattended

If you need to log in using a non-interactive environment, you can use service account credentials instead. Either
set the location of the credentials as the `JSCTL_CREDENTIALS` environment variable or provide the location via the
`--credentials` flag when calling `jsctl auth login`.

```shell
jsctl auth login --credentials /path/to/credentials.json
```

### Configuration

Once authenticated, select your organization using the `jsctl config set` command. The organization you select will be
used for subsequent commands

```shell
jsctl config set organization my-organization
> Your organization has been changed to my-organization
```

You can view which organizations you belong to using the `jsctl organizations list` command.

### Clusters

#### Connect Clusters

Once you've selected an organization, you can install the agent using the `jsctl clusters connect` command. This command
applies the YAML required to install an agent in your cluster. This uses your current kubernetes context as the target
for the deployment.

```shell
jsctl clusters connect my-cluster
```

Otherwise, you can write the output to a file and use it in your GitOps workflow:

```shell
jsctl clusters connect --stdout my-cluster >> agent.yaml
```

##### Custom image registry

If you want to use an alternative image registry for the agent image, you can specify the `--registry` flag that
allows you to change it:

```shell
jsctl clusters connect --registry my.exampleregistry.com
```

This will produce image names like `my.exampleregistry.com/preflight`. Currently, it is assumed images will have the
same name and tags.

#### List Clusters

To see all the clusters currently connected to the control-plane for an organization, you can use the `jsctl clusters list`
command:

```shell
jsctl clusters list
```

This produces a list of clusters and their last known update time. You can provide the `--json` flag to produce the list
as a JSON array. This could then be piped into a tool like `jq` for further processing.

#### Delete Clusters

You can remove a cluster from the control-plane using the `jsctl clusters delete` command and providing the cluster
name as the first argument:

```shell
jsctl clusters delete my-cluster
```

You will be prompted for confirmation for cluster deletion if the given response is anything except `y` or `Y` the deletion
is cancelled. If you do not want to confirm your choice, provide the `--force` flag:

```shell
jsctl clusters delete --force my-cluster
```

#### View Cluster

You can use the `jsctl clusters view` command to open your browser and navigate to the certificate inventory view within
Jetstack Secure for a chosen cluster:

```shell
jsctl clusters view my-cluster
```

In a non-interactive environment, or if the browser cannot be opened, the URL to visit will be written to the terminal
output.

### Users

#### List users

To list all users in your organization, you can use the `jstcl users list` command.  You can provide the `--json` flag
to produce the list as a JSON array. This could then be piped into a tool like `jq` for further processing.

```shell
jsctl users list
```

#### Add Users

To add a user to your organization, you can use the `jsctl users add` command and provide their email address. By default,
users will be created as members. You can provide the `--admin` flag to create the user as an administrator of your
organization.

```shell
jsctl users add [--admin] test@example.com
```

You can view the users within an organization using the `jsctl users list` command.

#### Remove Users

To remove a user from your organization, you can use the `jsctl users remove` command and provide their email address.

```shell
jsctl users remove test@example.com
```

You will be prompted for confirmation for user removal. If the given response is anything except `y` or `Y` the removal
is cancelled. If you do not want to confirm your choice, provide the `--force` flag:

```shell
jsctl users remove --force test@example.com
```

### Operator

#### Install the Operator

To install the Jetstack Operator, you can use the `jsctl operator deploy` command. It will apply the manifests required
to run the operator directly to your current kubernetes context. You will need to have obtained your secret key file for
authenticating with the Jetstack container registry and provide it to the command via the `--credentials` flag.

```shell
jsctl operator deploy --credentials /path/to/secret.json
```

To just obtain the manifests, provide the `--stdout` flag:

```shell
jsctl operator deploy --stdout --credentials /path/to/secret.json >> operator.yaml
```

By default, it will install the latest version of the operator. You can specify a specific version using the `--version`
flag:

```shell
jsctl operator deploy --credentials /path/to/secret.json --version v0.0.1-alpha.0
```

To view all available versions of the operator to install, you can use the `jsctl operator versions` command, which outputs
the versions in order from oldest to newest.

##### Custom image registry

If you want to use an alternative image registry for the operator image, you can specify the `--registry` flag that
allows you to change it:

```shell
jsctl operator deploy --registry my.exampleregistry.com
```

This will produce image names like `my.exampleregistry.com/js-operator`. Currently, it is assumed images will have the
same name and tags.

#### Create an installation

To create an `Installation` resource that is consumed by the operator, you can use the `jsctl operator installations apply`
command which will apply the resource directly to your current kubernetes context. On its own, this will install a
high-availability cert-manager deployment within your cluster with 2 replicas.

```shell
jsctl operator installations apply
```

To modify the number of cert-manager replicas, use the `--cert-manager-replicas` flag:

```shell
jsctl operator installations apply --cert-manager-replicas 3
```

To just output the YAML of the `Installation` resource, provide the `--stdout` flag:

```shell
jsctl operator installations apply --stdout
```

##### Install the cert-manager CSI driver

You can also provide the `--csi-driver` flag to include the installation of the [Cert Manager CSI Driver](https://github.com/cert-manager/csi-driver)
in your cluster:

```shell
jsctl operator installations apply --csi-driver
```

##### Install the SPIFFE CSI Driver

You can also provide the `--csi-driver-spiffe` flag to include the installation of the [SPIFFE CSI Driver](https://github.com/cert-manager/csi-driver-spiffe)
in your cluster:

```shell
jsctl operator installations apply --csi-driver-spiffe
```

By default, this deploys the SPIFFE CSI driver in a high-availability configuration, using two replicas. To change the
number of replicas you can use the `--csi-driver-spiffe-replicas` flag:

```shell
jsctl operator installations apply \
  --csi-driver-spiffe \
  --csi-driver-spiffe-replicas 3
```

##### Install Istio CSR

You can also provide the `--istio-csr` flag to include the installation of the [Istio CSR](https://github.com/cert-manager/istio-csr)
in your cluster:

```shell
jsctl operator installations apply --istio-csr
```

You must also configure the Istio CSR to use one of your issuers using the `--istio-csr-issuer` flag:

```shell
jsctl operator installations apply \
  --istio-csr \
  --istio-csr-issuer my-issuer
```

Once applied, you will need to modify/create the `IstioOperator` custom resource, configured to use istio-csr. This
configuration differs based on the Istio version you're using. You can see configuration examples [here](https://github.com/cert-manager/istio-csr/tree/main/hack)

By default, this deploys Istio CSR in a high-availability configuration, using two replicas. To change the number
of replicas you can use the `--istio-csr-replicas` flag:

```shell
jsctl operator installations apply \
  --istio-csr \
  --istio-csr-issuer my-issuer \
  --istio-csr-replicas 3
```

##### Checking component status

You can use the `jsctl operator installations status` command to check the status of all components installed by
the operator:

```shell
jsctl operator installations status
```

You can also provide the `--json` flag to get the output in JSON format:

```shell
jsctl operator installations status --json
```

## Development

- This repository depends on the private https://github.com/jetstack/js-operator Go module.
  To pull this module via Go you might need to set `GOPRIVATE` env var i.e `GOPRIVATE="github.com/jetstack/*" go get -u`
- `jsctl` writes configuration (current organization) to a local file, on UNIX the path will likely be `~/.config/jsctl/config.json`

## Attributions

When this project was made public commit history was wiped. The current
maintainers of the project are:

* [Irbe Krumina](https://github.com/irbekrm)
* [Charlie Egan](https://github.com/charlieegan3)

The original author of the project was
[David Bond](https://github.com/davidsbond).
[Mathias Gees](https://github.com/MattiasGees) has also contributed to the
project.
