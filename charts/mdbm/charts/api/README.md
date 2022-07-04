# MongoDB Operations API

This chart installs MongoDB Operations API deployment on a Kubernetes cluster
using the Helm package manager.

## Installing the chart

To install the chart with the release name `mdbm-api`:

```
$ helm install --name mdbm-api --namespace mdbm .
```

The command deploys MongoDB Operations API on the Kubernetes cluster in the
default configuration. The [configuration](#configuration) section lists the
parameters that can be configured during installation.

## Uninstalling the chart

To uninstall/delete the `mdbm-api` deployment:

```
$ helm delete --purge mdbm-api
```

The command removes all the Kubernetes components associated with the chart and
deletes the release.

## Configuration

**TODO:** Document configuration options
