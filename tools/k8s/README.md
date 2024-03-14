# Determined + Kubernetes Developer Guide

This tooling exists to run Determined backed by Kubernetes, but outside of
Kubernetes.  The reason for this is rapid development.  Each change to the
`determined-master` can be tested with a quick recompile and rerun.

## Step 1: Get a working kubernetes cluster

To use these files, you'll need a working Kubernetes cluster (not included).
I (rb) highly recommend [kind](https://github.com/kubernetes-sigs/kind), but
[minikube](https://minikube.sigs.k8s.io/docs/) is also popular.

If you want to configure a cloud cluster, these steps should work, but keep in
mind that you'll need your `determined-master` instance to be accessible from
all of your pods.  That means either port forwarding, or following these steps
on some cloud machine.

If you go the GKE cloude route, keep in mind that some GKE configurations
create an API server that is not generally accessible.

## Step 2: Make sure your kubeconfig is correct.

For example, when using Minikube everything should "just work" since `minikube start` mucks with your kubeconfig. When using GKE, you need to run something like `gcloud container clusters get-credentials ...`.

## Step 2: Run `determined-master` with a special `devcluster.yaml`

Read `tools/k8s/devcluster.yaml` so you know what it's doing, then run it:

```sh
devcluster -c tools/k8s/devcluster.yaml
```
