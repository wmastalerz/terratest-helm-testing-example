# MongoDB Operator

This chart installs MongoDB Operator deployment on a Kubernetes cluster using
the Helm package manager.

## Installing the chart

To install the chart with the release name `mdbm`:

```
$ helm install --name mdbm --namespace mdbm .
```

The command deploys MongoDB Operator on the Kubernetes cluster in the default
configuration. The [configuration](#configuration) section lists the parameters
that can be configured during installation.

## Uninstalling the chart

To uninstall/delete the `mdbm` deployment:

```
$ helm delete --purge mdbm
```

The command removes all the Kubernetes components associated with the chart and
deletes the release.

CRDs created by this chart are not removed by default and should be manually
cleaned up:

```
$ kubectl delete crd mongodbs.mongodb.samsung.com
$ kubectl delete crd mongodbreplicaset.mongodb.samsung.com
$ kubectl delete crd mongodbbackup.mongodb.samsung.com
$ kubectl delete crd pmmservers.mongodb.samsung.com
$ kubectl delete crd mongodbcontrollerruntimeconfigs.mongodb.samsung.com
```

## Configuration

**TODO:** Document configuration options
