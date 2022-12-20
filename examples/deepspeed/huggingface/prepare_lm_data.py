import argparse
import logging
from pathlib import Path
from typing import Union
from functools import partial

import transformers
from datasets import load_dataset, Dataset, DatasetDict, IterableDatasetDict

logger = logging.getLogger("prepare_data")


def get_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()

    # Required args
    parser.add_argument("--dataset_name",
                        type=str,
                        required=True,
                        help="Path argument to pass to HuggingFace ``datasets.load_dataset``")
    parser.add_argument("--processed_dataset_destination",
                        type=str,
                        required=True,
                        help="Path to directory where the preprocessed dataset will be saved.")
    parser.add_argument("--tokenizer_name",
                        type=str,
                        required=True,
                        help="Path to pretrained model or model identifier from huggingface.co/models")

    # Optional Dataset args
    parser.add_argument("--dataset_config_name",
                        type=str,
                        default=None,
                        help="The name of the dataset configuration to pass to HuggingFace ``datasets.load_dataset``.")
    parser.add_argument("--validation_split_percentage",
                        type=float,
                        default=None,
                        help="This is used to create a validation split from the training data when a dataset does " +
                             "not have a predefined validation split.")
    parser.add_argument("--dataset_cache_dir",
                        type=Path,
                        default=None,
                        help="Path to the directory to be used as a cache when downloading the dataset. " +
                             "A previously cached dataset will be used instead of redownloaded.")

    # Optional tokenizer args
    parser.add_argument("--tokenizer_cache_dir",
                        type=Path,
                        default=None,
                        help="Path to the directory to be used as a cache when downloading the tokenizer. " +
                             "A previously cached tokenizer will be used instead of redownloaded.")
    parser.add_argument("--tokenizer_revision",
                        type=str,
                        default="main",
                        help="The specific model version to use (can be a branch name, tag name or commit id)")

    # Optional preprocessing args
    parser.add_argument("--preprocessing_num_workers",
                        type=int,
                        default=1,
                        help="Number of workers to use when tokenizing the dataset")
    parser.add_argument("--preprocessing_batch_size",
                        type=int,
                        default=1000,
                        help="Batch size of texts when preprocessing. Defaults to 1000")
    parser.add_argument("--max_seq_len",
                        type=int,
                        default=1024,
                        help="Max sequence length for each tokenized input. Defaults to 1024")
    parser.add_argument("--overwrite_cache",
                        action="store_true",
                        help="Flag to specify if the preprocessing cache should be overwritten.")

    return parser.parse_args()


def load_raw_dataset(dataset_name: str,
                     dataset_config_name: str,
                     dataset_cache_dir: str,
                     validation_split_percentage: float) -> Union[DatasetDict, IterableDatasetDict]:
    datasets = load_dataset(dataset_name,
                            dataset_config_name,
                            cache_dir=dataset_cache_dir)

    assert hasattr(datasets, "keys"), "Expected a dictionary of datasets."

    if "validation" not in datasets.keys():
        assert validation_split_percentage is not None, \
            "Validation split not provided by this huggingface dataset. " \
            "Please specify validation_split_percentage in data_config for use to create validation set."

        datasets["validation"] = load_dataset(dataset_name,
                                              dataset_config_name,
                                              split=f"train[:{validation_split_percentage}%]")
        datasets["train"] = load_dataset(dataset_name,
                                         dataset_config_name,
                                         split=f"train[{validation_split_percentage}%:]")

    return datasets


def build_datasets(raw_ds: Union[DatasetDict, IterableDatasetDict],
                   tokenizer: transformers.PreTrainedTokenizer,
                   preprocessing_num_workers: int,
                   preprocessing_batch_size: int,
                   max_seq_len: int,
                   overwrite_cache: bool) -> Union[Dataset, DatasetDict]:
    column_names = raw_ds["train"].column_names
    text_column_name = "text" if "text" in column_names else column_names[0]

    def tokenize_func(tokenizer, examples):
        return tokenizer(examples[text_column_name])

    tokenized_datasets = raw_ds.map(partial(tokenize_func, tokenizer),
                                    batched=True,
                                    num_proc=preprocessing_num_workers,
                                    remove_columns=column_names,
                                    load_from_cache_file=not overwrite_cache)

    if tokenizer.model_max_length < max_seq_len:
        logger.warning("The max_seq_len passed is larger than the maximum length for the model. Using max_seq_len="
                       f"{tokenizer.model_max_length}.")
        max_seq_len = tokenizer.model_max_length

    # Main data processing function that will concatenate all texts from our dataset and generate chunks of max_seq_len
    def group_texts(examples):
        # Concatenate all texts.
        concatenated_examples = {k: sum(examples[k], []) for k in examples.keys()}
        total_length = len(concatenated_examples[list(examples.keys())[0]])
        # We drop the small remainder. We could add padding if the model supported it instead of this drop.
        # You can customize this part to your needs
        total_length = (total_length // max_seq_len) * max_seq_len
        # Split chunks of max_len
        result = {key: [text[i: i+max_seq_len] for i in range(0, total_length, max_seq_len)]
                  for key, text in concatenated_examples.items()}
        result["labels"] = result["input_ids"].copy()
        return result

    # Note that with `batched=True`, this map processes 1,000 texts together,
    # so group_texts throws away a remainder for each of these groups of 1,000 texts.
    # You can adjust that batch_size here but a higher value might be slower to preprocess.
    lm_datasets = tokenized_datasets.map(group_texts,
                                         batched=True,
                                         batch_size=preprocessing_batch_size,
                                         num_proc=preprocessing_num_workers,
                                         load_from_cache_file=not overwrite_cache)

    return lm_datasets


def main() -> None:
    args = get_args()

    tokenizer = transformers.AutoTokenizer.from_pretrained(pretrained_model_name_or_path=args.tokenizer_name,
                                                           cache_dir=args.tokenizer_cache_dir,
                                                           revision=args.tokenizer_revision,
                                                           use_fast=True)
    raw_ds = load_raw_dataset(args.dataset_name,
                              args.dataset_config_name,
                              args.dataset_cache_dir,
                              args.validation_split_percentage)
    lm_datasets = build_datasets(raw_ds,
                                 tokenizer,
                                 args.preprocessing_num_workers,
                                 args.preprocessing_batch_size,
                                 args.max_seq_len,
                                 args.overwrite_cache)

    lm_datasets.save_to_disk(args.processed_dataset_destination)


if __name__ == "__main__":
    main()
