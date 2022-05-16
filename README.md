# cluster-identiy-controller

A k8s operator which finds the cluster identity and writes the information to a `configmap`

## Short introduction

This operator is made to find the cluster identity and write that information into a `configmap` so that services that depend on that information has it available.

## How it works

The operator monitors all namespaces in the cluster it is installed into and looks for the annotation `config.lunar.tech/cluster-identity-inject: "true"`.
For those namespaces, if the operator can identify what cluster it is running in, it will create and manage a `configmap` called `cluster-identity`.

## Supported Clusters

The operators has a list of strategies which are tried, one at a time. If one strategy it successful, then it is used to populate the `configmap`.

Currently, the following strategies are supported:

- coreDNSClusterNameStrategy: Checks the core DNS autoscaler pod environment variable: `KUBERNETES_PORT_443_TCP_ADDR`
- kubeControllerStrategy: Checks the kube controller pod definition.
- nodeLabelStrategy: Check nodes for a `clusterName` label

## Releasing

Releases are automated via Release Drafter. Commits to the default branch are automatically picked up and added the a draft release. When ready, publish the draft release.
