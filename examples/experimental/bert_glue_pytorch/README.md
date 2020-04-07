*This folder contains the required files and the example code to use Huggingface's run_glue.py script with Determined.*
## The file version can be found on [Huggingface's run_glue example](https://github.com/huggingface/transformers/blob/v2.2.1/examples/run_glue.py)

For this implementation, we removed the system configuration since Determined will be managing this. The core functionality of the original script remains unchanged.

### Folders and Files
* **model_def.py**: Contains the core code for the model. This includes building and compiling the model.  
* **data.py**: Contains the data loading and preparation code for the model.
* **const.yaml**: Contains the configuration for the experiment. This is also where you can set the flags used in the original script.  
* **const_multi_gpu.yaml**: Same as const.yaml, but instead uses multiple GPUs.
* **download_glue_data**: Contains the code to download glue data based on tasks. This script is from [W4ngatang](https://gist.github.com/W4ngatang/60c2bdb54d156a41194446737ce03e2e) which is reference on the [Huggingface's GLUE page](https://github.com/huggingface/transformers/tree/v2.2.1/examples#glue)

### Data
   The data used for this script was fetched based on Huggingface's [GLUE page](https://github.com/huggingface/transformers/tree/v2.2.1/examples#glue).

   There are two options to access the data:

   * Data can be downloaded from [Huggingface's GLUE page](https://github.com/huggingface/transformers/tree/v2.2.1/examples#glue). The data needs to be available at the same path on all of the agents where Determined is running. The absolute file path will need to be uncommented and updated in the const.yaml and const_multi_gpu.yaml under bind_mounts. Then when running, Determined will look for the absolute path to mount to the container_path assigned in the yaml files.

   * If the dataset does not exist during runtime, and download_data in the yaml files is set to True, the project will automatically download the data and save to the provided data_dir directory for future use.

   This script can be used to run BERT, XLM, XLNet, and RoBERTa on multiple GLUE tasks, such as MRPC. The full list and their median results can be found on the link above.

### To Run
   *Prerequisites*:  
      Installation instruction found under `docs/install-admin.html` or at [Determined installation page](https://docs.determined.ai/latest/index.html).

   After configuring the settings in const.yaml. Run the following command:
     `det -m <master host:port> experiment create -f const.yaml . `
