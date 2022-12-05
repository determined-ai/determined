# Determined.ai helm chart repository

## Usage

Enable repository:
`helm repo add determined-ai https://helm.determined.ai/`

List repos:
`helm repo list`

Get updates to repo (helm doesn't automatically update):
`helm repo update`

Show current version of determined-ai in repo:
`helm search repo determined`

Install current version with completely minimal defaults:
`helm install --generate-name determined-ai/determined --set maxSlotsPerPod=1`

More information on values that can be set in the helm chart:
<https://docs.determined.ai/latest/reference/reference-deploy/config/helm-config-reference.html>

More information on installing under Kubernetes:
<https://docs.determined.ai/latest/cluster-setup-guide/deploy-cluster/sysadmin-deploy-on-k8s/install-on-kubernetes.html>
