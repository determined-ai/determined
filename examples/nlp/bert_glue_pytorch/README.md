# PyTorch Fine-Tuning BERT on GLUE Text Classification Example

This example shows how to fine-tune BERT on the GLUE text classification dataset using
Determined's PyTorch API. This example is adapted from [HuggingFace's run_glue
example](https://github.com/huggingface/transformers/blob/v2.2.1/examples/run_glue.py).

## Files
* **model_def.py**: The core code for the model. This includes building and compiling the model.
* **data.py**: The data loading and preparation code for the model.
* **startup-hook.sh**: Additional dependencies that Determined will automatically install into each container for this experiment.

### Configuration Files
* **const.yaml**: Train the model with constant hyperparameter values.
* **distributed.yaml**: Same as `const.yaml`, but trains the model with multiple GPUs (distributed training).

## Data
The GLUE datasets are downloaded with the HuggingFace datasets library.

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
