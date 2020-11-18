# PyTorch Fine-Tuning ALBERT on SQuAD 2.0 Question-Answering Example

This example shows how to fine-tune the largest variant of ALBERT on the SQuAD 2.0 question-answering dataset using
Determined's PyTorch API. This example is adapted from [Huggingface's SQuAD
example](https://github.com/huggingface/transformers/blob/master/examples/question-answering/run_squad.py).

## Files
* **model_def.py**: The core code for the model. This includes building and compiling the model.
* **data.py**: The data loading and preparation code for the model.
* **constants.py**: Constant references to models able to run on SQuAD.
* **startup-hook.sh**: Additional dependencies that Determined will automatically install into each container for this experiment.

### Configuration Files
* **const.yaml**: Train the model on 1 GPU.
* **distributed_8gpu.yaml**: Train the model on 8 GPUs (distributed training) while maintaining the same accuracy
* **distributed_64gpu.yaml**: Train the model on 64 GPUs (distributed training) while using the RAdam optimizer. 

These should run on any GPUs with sufficient memory, but they were optimized on V100-16GB GPUs with 25 Gbit/s networking.


## Results

Accuracy and throughput table

Scaling efficiency table

1 GPU
Throughput = 2 img/s
Time-to-process 2 epochs = 36.67 hours

8 GPUS
Throughput = 15.8 img/s
Time-to-process 2 epochs = 4.64 hours
After 2 epochs, model should hit 85.76/88.87 F1/EM

64 GPUs
Throughput = 91.43 img/s
Time-to-process 2 epochs = 0.8 hours
After 2 epochs, model should hit 86.24/89.06 F1/EM


### Caching

Extracting features from the dataset is quite time-consuming so this code will cache the extracted features to a file and not re-extract the features if that file is present. With containers, files that are saved to the container's file system are deleted when the container closes, so to reuse the cache file across experiments, you will need to set up a `bind_mount` in the experiment configuration, which allows the container to write to the host machine's file system.  

This caching works when you are running repeated experiments with the same agents, but in a cloud environment when you want to shut down VMs when they aren't in use, the cache will be emprt on any newly created VMs. To avoid this, you can have the cloud VMs use a network attached filesystem (e.g. EFS or FSx for Lustre on AWS) and bind mount a directory on the filesystem (for more details, see [our docs](https://docs.determined.ai/latest/tutorials/data-access.html#distributed-file-system))

All of the experiment configs in this directory set up `bind_mounts`. In our setup, a network file system is available on the VM at `/home/ubuntu/dtrain-fsx` and in the experiment we bind mount `/home/ubuntu/dtrain-fsx/albert-cache` to `/mnt/data`. This means that when the model code writes to `/mnt/data/`, it will be writing to the network file system. 

In order for the code to know where to save and look for the cache file, make sure to set the `data.use_bind_mount` and `data.bind_mount_path` fields correctly in the experiment configuration.

### Data
The data used for this script was fetched based on Huggingface's [SQuAD page](https://github.com/huggingface/transformers/tree/master/examples/question-answering).

The data will be automatically downloaded and saved before training. If you use a `bind_mount`, the data will be saved between experiments and will not need to be downloaded again.

## To Run
If you have not yet installed Determined, installation instructions can be found
under `docs/install-admin.html` or at https://docs.determined.ai/latest/index.html

Run the following command: `det -m <master host:port> experiment create -f
const.yaml .`. The other configurations can be run by specifying the appropriate
configuration file in place of `const.yaml`.


