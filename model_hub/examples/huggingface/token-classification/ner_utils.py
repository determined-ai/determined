from typing import Any, Dict, List

import datasets
import numpy as np
from attrdict import AttrDict
from model_hub import huggingface as hf
from seqeval import metrics


def compute_metrics(label_list: List[Any], predictions: np.ndarray, labels: np.ndarray) -> Dict:
    predictions = np.argmax(predictions, axis=2)

    # Remove ignored index (special tokens)
    true_predictions = [
        [label_list[pr] for (pr, la) in zip(prediction, label) if la != -100]
        for prediction, label in zip(predictions, labels)
    ]
    true_labels = [
        [label_list[la] for (pr, la) in zip(prediction, label) if la != -100]
        for prediction, label in zip(predictions, labels)
    ]

    return {
        "accuracy_score": metrics.accuracy_score(true_labels, true_predictions),
        "precision": metrics.precision_score(true_labels, true_predictions),
        "recall": metrics.recall_score(true_labels, true_predictions),
        "f1": metrics.f1_score(true_labels, true_predictions),
    }


def get_dataset_metadata(raw_datasets, hparams):
    column_names = raw_datasets["train"].column_names
    features = raw_datasets["train"].features
    text_column_name = "tokens" if "tokens" in column_names else column_names[0]
    label_column_name = (
        "{}_tags".format(hparams.finetuning_task)
        if "{}_tags".format(hparams.finetuning_task) in column_names
        else column_names[1]
    )

    # Setup labels
    if isinstance(features[label_column_name].feature, datasets.ClassLabel):
        label_list = features[label_column_name].feature.names
        # No need to convert the labels since they are already ints.
        label_to_id = {i: i for i in range(len(label_list))}
    else:
        label_list = hf.get_label_list(raw_datasets["train"][label_column_name])
        label_to_id = {l: i for i, l in enumerate(label_list)}

    return AttrDict(
        {
            "num_labels": len(label_list),
            "label_list": label_list,
            "text_column_name": text_column_name,
            "label_column_name": label_column_name,
            "label_to_id": label_to_id,
        }
    )


def build_tokenized_datasets(
    raw_datasets,
    model,
    data_config,
    tokenizer,
    text_column_name,
    label_column_name,
    label_to_id,
):
    padding = "max_length" if data_config.pad_to_max_length else False

    def tokenize_and_align_labels(
        examples,
    ):
        tokenized_inputs = tokenizer(
            examples[text_column_name],
            padding=padding,
            truncation=True,
            # We use this argument because the texts in our dataset are lists of words
            # (with a label for each word).
            is_split_into_words=True,
        )
        labels = []
        for i, label in enumerate(examples[label_column_name]):
            word_ids = tokenized_inputs.word_ids(batch_index=i)
            previous_word_idx = None
            label_ids = []
            for word_idx in word_ids:
                # Special tokens have a word id that is None. We set the label to -100 so they
                # are automatically ignored in the loss function.
                if word_idx is None:
                    label_ids.append(-100)
                # We set the label for the first token of each word.
                elif word_idx != previous_word_idx:
                    label_ids.append(label_to_id[label[word_idx]])
                # For the other tokens in a word, we set the label to either the current label
                # or -100, depending on the label_all_tokens flag.
                else:
                    label_ids.append(
                        label_to_id[label[word_idx]] if data_config.label_all_tokens else -100
                    )
                previous_word_idx = word_idx

            labels.append(label_ids)
        tokenized_inputs["labels"] = labels
        return tokenized_inputs

    tokenized_datasets = raw_datasets.map(
        tokenize_and_align_labels,
        num_proc=data_config.preprocessing_num_workers,
        load_from_cache_file=not data_config.overwrite_cache,
        batched=True,
    )

    for _, data in tokenized_datasets.items():
        hf.remove_unused_columns(model, data)

    return tokenized_datasets
