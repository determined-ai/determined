### Running kubectl and helm Commands on Remote Private GKE Cluster

If you have a shared cluster with a firewall'd VPC running in a GCP project and would like to run
`kubectl` or `helm` commands on that cluster, you need to establish a secure connection to the
cluster's control plane. To do that, run the following command from `tools/k8s/shared_cluster`:

```
source cluster_connect.sh <path-to-exports-file>
```

You should use `exports/ci_cluster.sh` as a template for the variables you need to define in order
to run the script.

So, for example, running `source cluster_connect.sh exports/ci_cluster.sh` would allow for the
successful run of commands like `kubectl cluster-info` when trying to get details about cluster
`$GKE_CLUSTER_NAME`.
