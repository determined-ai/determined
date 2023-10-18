# DeepSpeed CPU Offloading example
This example is adapted from the 
[CIFAR example in the DeepSpeedExamples](https://github.com/microsoft/DeepSpeedExamples/tree/master/cifar) 
repository. It is intended to show how to configure 
[ZeRO Stage 3 with CPU offloading](https://www.deepspeed.ai/tutorials/zero/) for a simple CNN network.

Compared to the original example, MoE is removed due to `AssertionError: MoE not supported with Stage 3`.

**IMPORTANT!** 
ZeRO Stage 3 enables training models that do not fit into GPU memory by offloading optimizer, parameters and gradients to CPU
(RAM memory). The assumption is that RAM memory >> GPU memory, however when running on an instance with limited RAM 
memory, it is possible to experience memory-related errors as there is not enough memory for offloading.
Read more about memory requirements at https://deepspeed.readthedocs.io/en/latest/memory.html and take a look ath the 
`Final notes` section at the end.


## Files
* **model_def.py**: The core code for the model implemented as DeepSpeedTrial. 
The key portion of the code is initializing DeepSpeed ZeRO in line 52:
```
with deepspeed.zero.Init():
    model = Net(self.args)
```
`deepspeed.zero.Init()` creates a context that accelerates model initialization and allows to allocate models that
exceed the size of local CPU/GPU memory/NVMe, but fit within the aggregated CPU/GPU memory/NVMe across all nodes. 

Initializing a model in the `deepspeed.zero.Init()` context has the following effects:
* allocates tensors to CPU or GPU memory or NVMe,
* converts floating points tensors to half precision,
* **immediately** partitions tensors among the group of data-parallel devices,
* _(optional)_ uses more memory-efficient Linear Layer implementation.

Note that even if you provide ZeRO Stage 3 configuration, but the model is not initialized in the context, the entire
model has to fit into CPU memory in order to be initialized correctly.


### Configuration Files
* **zero_stages_3_no_offload.yaml**: Determined config to train the model with `ds_config_no_offload.json`.
* **ds_config_no_offload.json**: The DeepSpeed config file with ZeRO turn off.
* **zero_stages_3_offload.yaml**: Determined config to train the model `ds_config_offload.json`.
* **ds_config_offload.json.yaml**: The DeepSpeed config file with ZeRO Stage 3 and CPU offloading.


## Devices
2 x Nvidia T4 16GB RAM:
  * AWS: g4dn.2xlarge


## Data
The CIFAR-10 dataset is downloaded from https://www.cs.toronto.edu/~kriz/cifar.html.


## To Run
If you have not yet installed Determined, installation instructions can be found
under `docs/install-admin.html` or at https://docs.determined.ai/latest/index.html

To observe OOM when running the model without ZeRO Stage 3 CPU offloading, run the following command: 
```
det experiment create zero_stages_3_no_offload.yaml .
``` 

To see how ZeRO Stage 3 CPU offloading allows for running a model that exceeds GPU memory, run the following command: 
```
det experiment create zero_stages_3_offload.yaml .
``` 


## Final notes
* If you are facing the following error, make sure that you are using the current version of deepspeed 
and our Deep Speed config `ds_config_offload.json.yaml`.
```
RuntimeError: weight should have at least three dimensions
```
* While ZeRO Infinity offers offloading to NVMe, currently [saving checkpoints is disabled by Deep Speed](https://github.com/microsoft/DeepSpeed/issues/2082), and hence 
NVMe offloading is also not supported by Determined.
* If your RAM memory is fully occupied by the offloaded data, you are very likely to experience the following error while saving the checkpoint:
```
[f35a2cc1] [rank=0] [INFO] [logging.py:69:log_dist] [Rank 0] Saving model checkpoint: /tmp/8596df39-c441-4259-a890-705a32f96326/model0/zero_pp_rank_0_mp_rank_00_model_states.pt
[f13fadd2] [rank=1] [INFO] [logging.py:69:log_dist] [Rank 1] Saving model checkpoint: /tmp/8596df39-c441-4259-a890-705a32f96326/model0/zero_pp_rank_1_mp_rank_00_model_states.pt
[f35a2cc1] 172.31.42.141: [INFO] [launch.py:178:sigkill_handler] Killing subprocess 190
[f35a2cc1] 172.31.36.1: [INFO] [launch.py:178:sigkill_handler] Killing subprocess 204
[f35a2cc1] 172.31.42.141: [ERROR] [launch.py:184:sigkill_handler] ['python3', '-m', 'determined.exec.pid_client', '/tmp/pid_server-67.64195e83-8b79-43e7-b650-ed71423daf3d.1', '--', 'python3', '-m', 'determined.launch.wrap_rank', 'RANK', '--', 'python3', '-m', 'determined.exec.harness', 'model_def_OOM_offload:CIFARTrial'] exits with return code = 247
[f35a2cc1] 172.31.36.1: [ERROR] [launch.py:184:sigkill_handler] ['python3', '-m', 'determined.exec.pid_client', '/tmp/pid_server-67.64195e83-8b79-43e7-b650-ed71423daf3d.1', '--', 'python3', '-m', 'determined.launch.wrap_rank', 'RANK', '--', 'python3', '-m', 'determined.exec.harness', 'model_def_OOM_offload:CIFARTrial'] exits with return code = 247
[f35a2cc1] pdsh@ip-172-31-36-1: 172.31.36.1: ssh exited with exit code 247
[f35a2cc1] pdsh@ip-172-31-36-1: 172.31.42.141: ssh exited with exit code 247
[f35a2cc1] resources failed with non-zero exit code: container failed with non-zero exit code: 247 (exit code 247)
INFO: forcibly killing allocation's remaining resources (reason: resources failed with non-zero exit code: container failed with non-zero exit code: 247 (exit code 247))
[f13fadd2] resources failed with non-zero exit code: container failed with non-zero exit code: 78 (exit code 78)
INFO: Trial (Experiment 67) was terminated: allocation failed: resources failed with non-zero exit code: container failed with non-zero exit code: 247 (exit code 247)
```
Possible solutions include:
* offloading either optimizer or params, but not both,
* training your model on a machine with more RAM memory,
* decreasing the network size.

Deepspeed is actively working on decreasing memory allocation for CPU offloading. Check the status [here](https://github.com/microsoft/DeepSpeed/issues/2003).