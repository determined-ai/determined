# DeepSpeed CPU Offloading example
This example is adapted from the 
[CIFAR example in the DeepSpeedExamples](https://github.com/microsoft/DeepSpeedExamples/tree/master/cifar) 
repository. It is intended to show how to configure [ZeRO Stage 3 with CPU offloading](https://www.deepspeed.ai/tutorials/zero/)
for a simple CNN network.


## Files
* **model_def_OOM_no_offload.py**: The core code for the model implemented as DeepSpeedTrial. Running this model without 
ZeRO Infinity CPU offload results in CUDA Out of Memory (OOM) error.
* **model_def_OOM_offload.py**: The same model architecture as in `model_def_OOM_no_offload.py`, however this time the 
model is running with ZeRO Stage 3 CPU offloading. 
The key difference is initializing DeepSpeed ZeRO in line 50:
```
with deepspeed.zero.Init():
    model = Net(self.args)
```

### Configuration Files
* **zero_stages_3_no_offload.yaml**: Determined config to train the model with `ds_config_no_offload.json`.
* **ds_config_no_offload.json**: The DeepSpeed config file with ZeRO turn off.
* **zero_stages_3_offload.yaml**: Determined config to train the model `ds_config_offload.json`.
* **ds_config_offload.json.yaml**: The DeepSpeed config file with ZeRO Stage 3 and CPU offloading.


### Other files
* **requirement.txt**: Contain necessary python packages
* **startup-hook.sh**: Install python packages and `pdsh` (required by DeepSpeed).


## Devices
2 x Nvidia T4 16GB RAM:
  * AWS: g4dn.xlarge
  * GCP: nvidia-tesla-t4


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
* If you are faced with the following error, make sure that you are using the current version of deepspeed 
and our Deep Speed config `ds_config_offload.json.yaml`.
```
RuntimeError: weight should have at least three dimensions
```
* While ZeRO Infinity offers offloading to NVMe, currently [saving checkpoints is disabled by Deep Speed](https://github.com/microsoft/DeepSpeed/issues/2082), and hence 
NVMe offloading is also not supported by Determined.