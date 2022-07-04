# Helm Chart test using Terratest

This repository contains code in order to test ACN charts.

The tests for those charts are located under the `test` directory. The `test` folder contains the corresponding terratest code
for testing each of the charts in the `charts` directory.

## Quickstart

### Prerequisites

To run the tests, you will need a working go install. See [here](https://golang.org/doc/install) for instructions on
installing go on to your platform. Make sure to use a version >=1.13.

In order to install necessary golang modules:
```
cd test
go mod init
go mod tidy
```

### Kubernetes cluster

Some of the tests (specifically the integration tests) require a Kubernetes cluster to run.
Need to install the helm client run the tests based on `helm install`.
Follow the [official guide](https://helm.sh/docs/intro/install/) for instructions on installing `helm`.

### Running the tests

To run the tests, first change directory to the `test` folder of this repository, and then use `go test` to run the tests:

```
cd test
go test -v .
```
