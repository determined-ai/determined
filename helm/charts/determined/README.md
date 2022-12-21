![determined](https://github.com/determined-ai/determined/blob/master/determined-logo.png?raw=true)

# Determined: Deep Learning Training Platform

[Determined](https://www.determined.ai/) is an open-source deep learning training platform that makes building models fast and easy. Determined can be deployed directly on CoreWeave Cloud by deploying this App. Once deployed, the platform consumes minimal resources and incurs minimal cost. No GPU resources are consumed unless experiments are running. This prevents resource waste and idle compute compared to running experiments in Virtual Machines. Determined on CoreWeave supports multi GPU and multi node distributed training out of the box. The platform comes with support for many popular frameworks, including [GPT-Neo-X](https://github.com/determined-ai/gpt-neox) to train/finetune the [EleutherAI 20B](https://blog.eleuther.ai/announcing-20b/) model.

The CoreWeave app allows for selection of a default region and hardware type for running experiments. These default selections allow users to submit experiments without any additional configuration. Storage volumes can also be attached, this allows for loading of training data and storage of checkpoints directly to a shared filesystem that can be mounted into Pods, Virtual Servers and File Browser (can be deployed from Apps).

## Demo

Watch our demo of installing and using Determined on CoreWeave Cloud:

[![Watch our demo on installing and using Determined on CoreWeave Cloud](https://github.com/coreweave/coreweave_determined/raw/coreweave/helm/charts/determined/coreweave-determinedvidprev-small.png)](https://youtu.be/0lH5clFoe5c)

## Pre-Requisites

- You will need object storage bucket on CoreWeave Object Storage for checkpoints. _Please contact CoreWeave support to request Object Storage access_ 
- Install the determined CLI via: https://docs.determined.ai/latest/interact/cli.html

## Installation

The Determined application comes pre-configured with most intallations values to successfuly deploy on CoreWeave Cloud.

| Helm Chart Config Value  | Description |
| ------------- | ------------- |
| Region | Region where you are deploying determined. This should match the region where you intend to execute most of your training workloads.  |
| Scheduler | Default Scheduler type for experiments and notebooks, can be overriden in experiment configuration. The standard scheduler is good for most use cases, however large distributed training jobs will want to use coscheduler to allow jobs to queue when resources to run all jobs in parallel are not available. (default: default-scheduler) |
| vCPU Request | Default number of vCPUs for experiments, can be overriden in experiment configuration. (default: 8) |
| Memory Request | Default memory allocation for experiemnts, can be overriden in experiment configuration. (default: 32Gi) |
| GPU Type | Default GPU type for experiments and notebooks, can be overriden in experiment configuration. (default: RTX_A5000) |
| Mounts | Optional default Persistent Volume mounts for experiments and notebooks, can be overriden in experiment configuration. |
| Bucket Name (S3) | CoreWeave Object Storage Bucket Name for checkpoints |
| Access Key (S3) | Access Key for Object Storage |
| Secret Key (S3) | Secret Key for Object Storage |

## Connecting to determined.ai Master

- Ensure that determined CLI is installed
- Run ```export DET_MASTER=<this value will be displayed in the post-installation notes>``` to access the master
- Run ```det experiment list``` and ensure your output looks similar to this:
```
ID   | Owner   | Name   | Parent ID   | State   | Progress   | Start Time   | End Time   | Resource Pool 
------+---------+--------+-------------+---------+------------+--------------+------------+-----------------
```

## Web UI
- The link to the Web UI will be presented in the post-installation notes
- The default username is ```admin``` and the password field is blank. You must edit this default password after deployment.

## Running Experiments

You can start by running any of the [examples](https://docs.determined.ai/latest/examples.html). Below are details on customizing training jobs. 

**IMPORTANT:**

```
# This is the number of GPUs there are per machine. Determined uses this information when scheduling
# multi-GPU tasks. Each multi-GPU (distributed training) task will be scheduled as a set of
# `slotsPerTask / maxSlotsPerPod` separate pods, with each pod assigned up to `maxSlotsPerPod` GPUs.
# Distributed tasks with sizes that are not divisible by `maxSlotsPerPod` are never scheduled. If
# you have a cluster of different size nodes (e.g., 4 and 8 GPUs per node), set `maxSlotsPerPod` to
# the greatest common divisor of all the sizes (4, in that case).
maxSlotsPerPod: 8
```

- In the configuration, ```slotsPerTask``` -> ```slots_per_trial```. Therefore, if you set ```slots_per_trial: 16```, two pods with 8 GPUs each will be spawned for the training workload.

### Running a custom training job

**Multi-node GPU training using default GPU**

This will run over two physical nodes with 8 GPUs each.

_Note the ```slots_per_trial: 16```_


```yaml
name: fashion_mnist_tf_keras_distributed
hyperparameters:
  global_batch_size: 256
  dense1: 128
resources:
  slots_per_trial: 16
records_per_epoch: 600
environment:
searcher:
  name: single
  metric: val_accuracy
  smaller_is_better: false
  max_length:
    epochs: 5
entrypoint: model_def:FashionMNISTTrial
```

**Multi-node GPU training using custom GPU selection**

This will run over four physical nodes with 8 GPUs each, on NVIDIA A40 GPUs in the ORD1 region.

_Note that per-GPU batch size =  ```global_batch_size // slots_per_trial = 16```_

```yaml
name: fashion_mnist_tf_keras_distributed
hyperparameters:
  global_batch_size: 512
  dense1: 128
resources:
  slots_per_trial: 32
records_per_epoch: 600
environment:
  pod_spec:
    spec:
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
              - matchExpressions:
                  - key: gpu.nvidia.com/class
                    operator: In
                    values:
                      - A40
                  - key: topology.kubernetes.io/region
                    operator: In
                    values:
                      - ORD1
searcher:
  name: single
  metric: val_accuracy
  smaller_is_better: false
  max_length:
    epochs: 5
entrypoint: model_def:FashionMNISTTrial
```

**Running multi-node GPU training using A100 NVLINK with Infiniband RDMA**

This will run over 12 physical nodes with 8 GPUs each, on A100 NVLINKs in the ORD1 region. Please note that custom Docker images are strongly recommended for proper Infiniband performance. CoreWeave provides a [repository with template Dockerfile](https://github.com/coreweave/nccl-tests) for customers to base their own images on.

_Note that per-GPU batch size =  ```global_batch_size // slots_per_trial = 16```_

```yaml
name: fashion_mnist_tf_keras_distributed
hyperparameters:
  global_batch_size: 1536
  dense1: 128
resources:
  slots_per_trial: 96
records_per_epoch: 600
environment:
  pod_spec:
    spec:
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
              - matchExpressions:
                  - key: gpu.nvidia.com/class
                    operator: In
                    values:
                      - A100_NVLINK
                  - key: topology.kubernetes.io/region
                    operator: In
                    values:
                      - ORD1
      containers:
        - name: determined-container
          resources:
            limits:
              rdma/ib: '1'
searcher:
  name: single
  metric: val_accuracy
  smaller_is_better: false
  max_length:
    epochs: 5
entrypoint: model_def:FashionMNISTTrial
```

## Mounting a PVC

- This allows you to deploy your training workload on a large amount of data that might not be optimal to fetch from object storage.
- Refer to the [CoreWeave Storage Documentation](https://docs.coreweave.com/coreweave-kubernetes/storage#available-storage-types) on how to allocate storage volumes
- Ensure that the storage volume is in the same region as the experiment you are running.
- You can add your PVC to your ```pod_spec```. Here is an example:

```yaml
name: fashion_mnist_tf_keras_distributed
hyperparameters:
  global_batch_size: 256
  dense1: 128
resources:
  slots_per_trial: 16
records_per_epoch: 600
environment:
  pod_spec:
    spec:
      volumes:
        - name: data
        persistentVolumeClaim:
          claimName: <PERSISTENT_VOLUME_CLAIM_NAME>
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
              - matchExpressions:
                  - key: gpu.nvidia.com/class
                    operator: In
                    values:
                      - A100_NVLINK
                  - key: topology.kubernetes.io/region
                    operator: In
                    values:
                      - ORD1
      containers:
        volumeMounts:
          - mountPath: "/data" # Path inside the container where you want the volume to mount to
            name: data
searcher:
  name: single
  metric: val_accuracy
  smaller_is_better: false
  max_length:
    epochs: 5
entrypoint: model_def:FashionMNISTTrial
```

## Useful Links

- [Distributed Training](https://docs.determined.ai/latest/training-distributed/index.html#multi-gpu-training)
- [Determined Github](https://github.com/determined-ai/determined)
- [Helm Chart Configuration Details](https://docs.determined.ai/latest/sysadmin-deploy-on-k8s/helm-config.html)
- [Determined Training APIS](https://docs.determined.ai/latest/training-apis/index.html)
- [Determined Cluster APIS](https://docs.determined.ai/latest/interact/index.html)



