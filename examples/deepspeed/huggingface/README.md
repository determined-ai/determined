# Huggingface

A full walkthrough of running this code is available on
the ["Finetuning HuggingFace LLMs with Determined AI and DeepSpeed" CoreWeave documentation page.](https://docs.coreweave.com/compass/determined-ai/finetuning-huggingface-llms-with-determined-ai-and-deepspeed)

This example builds upon
the [model hub huggingface examples](https://github.com/coreweave/coreweave_determined/tree/master/model_hub/examples/huggingface)
by leveraging [DeepSpeed](https://www.deepspeed.ai/) which boosts the performance of distributed training.

To use DeepSpeed with DeterminedAI, you must use
the [`DeepSpeedTrial`](https://docs.determined.ai/latest/training/apis-howto/deepspeed/deepspeed.html#deepspeed-api).

## Files
### Source code
 - **ds_config.json**: Contains the config that is passed to DeepSpeed.
 - **lm_trial.py**: The `DeepSpeedTrial` definition for language modeling.
 - **prepare_lm_data.py**: Script that preprocesses huggingface text datasets for the trials
 - **Dockerfile**: Defines the docker container that will be used in the trials

### Experiment configuration files
 - **opt125m_single.yml**: Performs a single trial that finetunes the OPT-125m model
 - **opt125m_search.yml**: Performs a hyperparameter search on the training micro batch size.

## Local Environment Setup
First create a local environment and install the requirements.
```
conda create --name=hf_deepspeed python=3.9
conda activate hf_deepspeed
pip install -r requirements.txt
```

To install the model_hub package, you need to build it from the source code (which is in this repo):
```
cd <repo_path>/model_hub
make build
pip install --find-links=./dist model-hub
```

## Preprocessing Datasets
You first must preprocess your dataset before finetuning. To do this we will use the `prepare_lm_data.py` script.

To be able to run the script via Determined it needs to be uploaded to the mounted PVC.

### Script Usage
```
usage: prepare_lm_data.py [-h] --dataset_name DATASET_NAME --processed_dataset_destination PROCESSED_DATASET_DESTINATION --tokenizer_name TOKENIZER_NAME [--dataset_config_name DATASET_CONFIG_NAME]
                          [--validation_split_percentage VALIDATION_SPLIT_PERCENTAGE] [--dataset_cache_dir DATASET_CACHE_DIR] [--tokenizer_cache_dir TOKENIZER_CACHE_DIR] [--tokenizer_revision TOKENIZER_REVISION]
                          [--preprocessing_num_workers PREPROCESSING_NUM_WORKERS] [--preprocessing_batch_size PREPROCESSING_BATCH_SIZE] [--max_seq_len MAX_SEQ_LEN] [--overwrite_cache]

optional arguments:
  -h, --help            show this help message and exit
  --dataset_name DATASET_NAME
                        Path argument to pass to HuggingFace ``datasets.load_dataset``
  --processed_dataset_destination PROCESSED_DATASET_DESTINATION
                        Path to directory where the preprocessed dataset will be saved.
  --tokenizer_name TOKENIZER_NAME
                        Path to pretrained model or model identifier from huggingface.co/models
  --dataset_config_name DATASET_CONFIG_NAME
                        The name of the dataset configuration to pass to HuggingFace ``datasets.load_dataset``.
  --validation_split_percentage VALIDATION_SPLIT_PERCENTAGE
                        This is used to create a validation split from the training data when a dataset does not have a predefined validation split.
  --dataset_cache_dir DATASET_CACHE_DIR
                        Path to the directory to be used as a cache when downloading the dataset. A previously cached dataset will be used instead of redownloaded.
  --tokenizer_cache_dir TOKENIZER_CACHE_DIR
                        Path to the directory to be used as a cache when downloading the tokenizer. A previously cached tokenizer will be used instead of redownloaded.
  --tokenizer_revision TOKENIZER_REVISION
                        The specific model version to use (can be a branch name, tag name or commit id)
  --preprocessing_num_workers PREPROCESSING_NUM_WORKERS
                        Number of workers to use when tokenizing the dataset
  --preprocessing_batch_size PREPROCESSING_BATCH_SIZE
                        Batch size of texts when preprocessing. Defaults to 1000
  --max_seq_len MAX_SEQ_LEN
                        Max sequence length for each tokenized input. Defaults to 1024
  --overwrite_cache     Flag to specify if the preprocessing cache should be overwritten.
```

### Example Call
```
det run 'python /mnt/finetune-opt/prepare_lm_data.py \ 
    --dataset_name wikitext \
    --dataset_config_name wikitext-103-raw-v1 \
    --processed_dataset_destination /mnt/finetune-opt/wikitext/processed \
    --tokenizer_name facebook/opt-125m \
    --dataset_cache_dir /mnt/finetune-opt/wikitext'
```

## Running Experiments

To run either of the included experiment configurations you must use the `det experiment create` command.
```
det experiment create opt125m_single.yml .
det experiment create opt125m_search.yml .
```
