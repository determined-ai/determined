# PyTorch Fine-Tuning BERT on GLUE Text Classification Example

This example shows how to fine-tune BERT on the GLUE text classification dataset using
Determined's PyTorch API. This example is adapted from [Huggingface's run_glue
example](https://github.com/huggingface/transformers/blob/v2.2.1/examples/run_glue.py).

## Files
* **model_def.py**: The core code for the model. This includes building and compiling the model.
* **data.py**: The data loading and preparation code for the model.
* **constants.py**: Constant references to models able to run on GLUE.
* **startup-hook.sh**: Additional dependencies that Determined will automatically install into each container for this experiment.
* **download_glue_data.py**: The code to download GLUE data based on tasks. This script is from [W4ngatang](https://gist.github.com/W4ngatang/60c2bdb54d156a41194446737ce03e2e) which is referenced on the [Huggingface's GLUE page](https://github.com/huggingface/transformers/tree/v2.2.1/examples#glue).

### Configuration Files
* **const.yaml**: Train the model with constant hyperparameter values.
* **distributed.yaml**: Same as `const.yaml`, but trains the model with multiple GPUs (distributed training).

## Data
The data used for this script was fetched based on Huggingface's [GLUE page](https://github.com/huggingface/transformers/tree/v2.2.1/examples#glue).

There are two options to access the data:

* You can download the training data from [Huggingface's GLUE page](https://github.com/huggingface/transformers/tree/v2.2.1/examples#glue). The data needs to be available at the same path on all of the agents where Determined is running. The absolute file path will need to be uncommented and updated in the `const.yaml` and `distributed.yaml` in the `bind_mounts` section. Then when running, Determined will look for the absolute path to mount to the container_path assigned in the yaml files.
* If the dataset does not exist during runtime and `download_data` in the yaml files is set to True, the project will automatically download the data and save to the provided `data_dir` directory for future use.

This script can be used to run BERT, XLM, XLNet, and RoBERTa on multiple GLUE tasks, such as MRPC. The full list and their median results can be found on the link above.

## To Run
If you have not yet installed Determined, installation instructions can be found
under `docs/install-admin.html` or at https://docs.determined.ai/latest/index.html

Run the following command: `det -m <master host:port> experiment create -f
const.yaml .`. The other configurations can be run by specifying the appropriate
configuration file in place of `const.yaml`.

## Results
Training the model with the hyperparameter settings in `const.yaml` should yield
a validation accuracy of ~83% on V100 GPUs.
