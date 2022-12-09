# GPT-NeoX
[GPT-NeoX](https://github.com/EleutherAI/gpt-neox) is an open-source fork of NVidia's 
[Megatron-LM](https://github.com/NVIDIA/Megatron-LM) repo that enables training large-language
models with 3D parallelism (data, pipeline, tensor parallelism) and DeepSpeed.  
This example implements Determined's DeepSpeedTrial interface for GPT-NeoX to demonstrate
the flexibility of our API as well as to enable training large-scale language models in Determined.

## Files
* **gpt2_trial.py**: The core code for the model. This includes building and compiling the model.
* **det_utils.py**: Helper functions and callbacks called within `gpt2_trial.py`.

### Configuration Files
* **zero1.yaml**: The Determined config file to train with 3D parallelism and ZeRO stage 1.
* **zero3.yaml**: The Determined config file to train with data and tensor parallelism and ZeRO stage 3.
* **gpt_neox_config/determined_cluster.yml**: The GPT-NeoX config file with paths for data files.

## Configuration
GPT-NeoX has it's own configuration system that is specified primarily through YAML files.  We 
include one such configuration [here](gpt_neox_config/determined_cluster.yml).  

If you want to add custom GPT-NeoX configs, you can place them in the `gpt_neox_config` directory 
and use them for your experiments by adding the config file to `hyperparameters.conf_file` in the 
experiment config.

## Docker Image
The docker image used by these experiments can be built using the [`Dockerfile`](Dockerfile).

## Data
The default dataset is the Enron Emails corpus build by running the `prepare_data.py` script 
in GPT-NeoX.  We have prebuilt this dataset into the docker image.  

## To Run
If you have not yet installed Determined, installation instructions can be found
under `docs/install-admin.html` or at https://docs.determined.ai/latest/index.html

Please make sure your cluster is setup to launch task containers with a shared filesystem 
mounted at `/run/determined/workdir/shared_fs`.  This is done by default for clusters created through
`det deploy gcp up`, `det deploy aws up --deployment-type efs`, `det deploy aws up --deployment-type fsx`.   

Once a cluster is available, run the following command: 
```
det experiment create zero1.yaml .
```

**Note:** You will need to run on GPUs that support fp16 training. 

## Results
Training with the provided configs for 10000 steps should yield a validation perplexity of ~2.3 on
the Enron Emails dataset. 
