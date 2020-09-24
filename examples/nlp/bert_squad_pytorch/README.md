# PyTorch Fine-Tuning BERT on SQuAD Question-Answering Example

This folder contains the required files and the example code to use Huggingface's run_squad.py script with Determined.
The file version can be found on [Huggingface's SQuAD example](https://github.com/huggingface/transformers/blob/master/examples/question-answering/run_squad.py)

### Files
* **model_def.py**: The core code for the model. This includes building and compiling the model.
* **data.py**: The data loading and preparation code for the model.
* **constants.py**: Constant references to models able to run on SQuAD. 
* **startup-hook.sh**: Extra dependencies that Determined is required to install. 
* **const.yaml**: A configuration file that trains the model with constant hyperparameter values. This is also where you can set the flags used in the original script. 
* **distributed.yaml**: Same as const.yaml, but instead uses multiple GPUs.

### Data
The data used for this script was fetched based on Huggingface's [SQuAD page](https://github.com/huggingface/transformers/tree/master/examples/question-answering).

There are two options to access the data:
    * Data can be downloaded from [Huggingface's SQuAD page](https://github.com/huggingface/transformers/tree/master/examples/question-answering). The data needs to be available at the same path on all of the agents where Determined is running. The absolute file path will need to be uncommented and updated in the const.yaml and distributed.yaml under bind_mounts. Then when running, Determined will look for the absolute path to mount to the container_path assigned in the yaml files.
    * If the dataset does not exist during runtime, and download_data in the yaml files is set to True, the project will automatically download the data and save to the provided data_dir directory for future use.

### To Run
Installation instructions can be found under `docs/install-admin.html` or at [Determined installation page](https://docs.determined.ai/latest/index.html). 
After configuring the settings in const.yaml, run the following command: `det -m <master host:port> experiment create -f const.yaml . `

### Results
After fine-tuning for 150 steps, model should achieve F1 = 88.52 per [Huggingface SQuAD](https://github.com/huggingface/transformers/tree/master/examples/question-answering).
