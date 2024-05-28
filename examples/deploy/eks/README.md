# Terraformed EKS cluster for Determined

This is an example terraform code to configure an EKS cluster to run Determined on.

Supported features:
- autoscaling via Karpenter,
- postgresql volume on EBS,
- shared fs on EFS.

Based on [original Karpenter example](https://github.com/terraform-aws-modules/terraform-aws-eks/tree/master/examples/karpenter)

## Prerequisites

- terraform
- helm
- aws CLI

## Installation

First, edit the `locals` section in `main.tf` to set your cluster name and AWS region.

```bash
$ terraform init
$ terraform apply -auto-approve
$ aws eks --region us-west-2 update-kubeconfig --name <CLUSTER NAME>
$ helm install determined determined-ai/determined --values values.yaml
```

## Teardown

Warning: shut down all the jobs in determined first.

```bash
$ helm uninstall determined
$ terraform destroy -auto-approve
```

## Future work

In the future, we may want to:
- Make the code configurable: currently, custom configurations will require changing the terraform code directly.
- Rework this code as `det deploy eks` utility.
- Switch from a postgres instance installed by helm and using an EBS volume to a terraform-provisioned RDS.
