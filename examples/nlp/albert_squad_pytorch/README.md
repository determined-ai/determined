# Fine-Tuning ALBERT on SQuAD 2.0




---
# OLD
# PyTorch Fine-Tuning BERT on SQuAD Question-Answering Example

This example shows how to fine-tune BERT on the SQuAD question-answering dataset using
Determined's PyTorch API. This example is adapted from [Huggingface's SQuAD
example](https://github.com/huggingface/transformers/blob/master/examples/question-answering/run_squad.py).

## Files
* **model_def.py**: The core code for the model. This includes building and compiling the model.
* **data.py**: The data loading and preparation code for the model.
* **constants.py**: Constant references to models able to run on SQuAD.
* **startup-hook.sh**: Additional dependencies that Determined will automatically install into each container for this experiment.

### Configuration Files
* **const.yaml**: Train the model with constant hyperparameter values.
* **distributed.yaml**: Same as `const.yaml`, but trains the model with multiple GPUs (distributed training).

### Data
The data used for this script was fetched based on Huggingface's [SQuAD page](https://github.com/huggingface/transformers/tree/master/examples/question-answering).

There are two options to access the data:

* You can download the training data from [Huggingface's SQuAD page](https://github.com/huggingface/transformers/tree/master/examples/question-answering). The data needs to be available at the same path on all of the agents where Determined is running. The absolute file path will need to be uncommented and updated in `const.yaml` and `distributed.yaml` in the `bind_mounts` section. Then when running, Determined will look for the absolute path to mount to the `container_path` specified in the yaml files.
* If the dataset does not exist during runtime and `download_data` in the yaml files is set to True, the project will automatically download the data and save to the provided `data_dir` directory for future use.

## To Run
If you have not yet installed Determined, installation instructions can be found
under `docs/install-admin.html` or at https://docs.determined.ai/latest/index.html

Run the following command: `det -m <master host:port> experiment create -f
const.yaml .`. The other configurations can be run by specifying the appropriate
configuration file in place of `const.yaml`.

## Results
Training the model with the hyperparameter settings in `const.yaml` should yield
F1 = 88.52 per [Huggingface SQuAD](https://github.com/huggingface/transformers/tree/master/examples/question-answering).
