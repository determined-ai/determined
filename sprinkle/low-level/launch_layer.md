# LAUNCH LAYER

The launch layer is a concept that already exists outside of Determined.

The goal of the launch layer is therefore to enable launch layers that exist outside of
Determined to also run in Determined, rather than to force all jobs to look alike.  That is,
we don't want to write the One Launch Layer To Rule Them All, we want to just let people bring
whatever launch layer they had before and still use it.

This has an important implication: the launch layer may differ between cluster and local training,
but the code launched by the launch layer should not have to change.

This the opposite goal as the context objects.  Context objects should be usable inside and outside
of Determined.  They may have different behaviors (pushing metrics to master or not, asking for
searcher operations or generating them from the config) but the same API in all cases.

- [horovodrun](https://horovod.readthedocs.io/en/stable/summary_include.html#running-horovod) (once on chief node):

    ```sh
    horovodrun -np 4 -H localhost:4 python train.py
    ```

- [tensorflow](https://www.tensorflow.org/guide/distributed_training#TF_CONFIG) (once per worker):

    ```python
    os.environ["TF_CONFIG"] = json.dumps({
        "cluster": {
            "worker": ["host1:port", "host2:port", "host3:port"],
            "ps": ["host4:port", "host5:port"]
        },
       "task": {"type": "worker", "index": 1}
    })
    ```

- [torch.distributed.launch](https://pytorch.org/docs/stable/distributed.html#launch-utility) (once per node, launches 1 worker per gpu):

    ```sh
    python -m torch.distributed.launch
        --nproc_per_node=NUM_GPUS_YOU_HAVE
        --nnodes=2
        --node_rank=0
        --master_addr="192.168.1.1"
        --master_port=1234
        YOUR_TRAINING_SCRIPT.py
    ```

- [torch\_xla](https://github.com/pytorch/xla/tree/master#start-distributed-training) (once on cheif?):

    ```sh
    python -m torch_xla.distributed.xla_dist
        --tpu=$TPU_POD_NAME
        --conda-env=torch-xla-1.7
        --env=XLA_USE_BF16=1
        -- python train.py
    ```

- [paddlepaddle](https://paddlepaddle.org.cn/documentation/docs/en/guides/06_distributed_training/cluster_quick_start_en.html) (once per node):

    ```sh
    # workers:
    python train.py
    # parameter servers:
    python -m paddle.distributed.launch_ps
        --worker_num 2
        --server_num 2
        train.py
    ```

- [torch elastic](https://pytorch.org/elastic/0.2.1/distributed.html) (once per node):
    ```sh
    python -m torchelastic.distributed.launch
        --nnodes=$NUM_NODES
        --nproc_per_node=$NUM_TRAINERS
        --rdzv_id=$JOB_ID
        --rdzv_backend=etcd
        --rdzv_endpoint=$ETCD_HOST:$ETCD_PORT
        YOUR_TRAINING_SCRIPT.py
    ```

## Examples

The following examples are sample ExperimentConfigs for configuring the launch layer:

- Basic usage:

    ```yaml
    #launch_layer: python3 -m determined.launch.auto_horovod
    entrypoint_script: python3 train.py
    ```

- Backwards compatibility with Trial-based training:

    ```yaml
    #launch_layer: python3 -m determined.launch.auto_horovod
    #entrypoint_script: python3 -m determined.exec.harness
    entrypoint: model_def:MyTrial
    ```

- Detectron2, which is based on pytorch's `DistributedDataParallel`:

    ```yaml
    launch_layer: python3 -m determined.launch.torch_distributed
    entrypoint_script: python3 train_detectron.py
    ```

- PyTorchLightning (horovod backend):

    ```yaml
    #launch_layer: python3 -m determined.launch.auto_horovod
    entrypoint_script: python3 train_lightning.py
    # user code sets accelerator=None or accelerator='horovod'
    ```

- PyTorchLightning (ddp backend):

    ```yaml
    launch_layer: null
    entrypoint_script: python3 train_lightning.py
    # user code sets accelerator=None or accelerator='ddp'/'ddp_spawn'/whatever
    ```

- Custom Profiling Tool:

    ```yaml
    launch_layer: python3 -m my.launch.layer
    entrypoint_script: my-custom-profiler --  python3 train_detectron.py
    ```
