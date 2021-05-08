"""
This example is largely based on the GLUE text-classification example in the huggingface
transformers library. The license for the transformer's library is reproduced below.

==================================================================================================

Copyright 2020 The HuggingFace Team. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
"""

import functools
import logging
from typing import Dict, Union

import attrdict
import datasets
import numpy as np
import transformers

import determined.pytorch as det_torch
import model_hub.huggingface as hf
import model_hub.utils as utils

task_to_keys = {
    "cola": ("sentence", None),
    "mnli": ("premise", "hypothesis"),
    "mrpc": ("sentence1", "sentence2"),
    "qnli": ("question", "sentence"),
    "qqp": ("question1", "question2"),
    "rte": ("sentence1", "sentence2"),
    "sst2": ("sentence", None),
    "stsb": ("sentence1", "sentence2"),
    "wnli": ("sentence1", "sentence2"),
}


class GLUETrial(hf.BaseTransformerTrial):
    def __init__(self, context: det_torch.PyTorchTrialContext) -> None:
        self.logger = logging.getLogger(__name__)
        self.hparams = attrdict.AttrDict(context.get_hparams())
        self.data_config = attrdict.AttrDict(context.get_data_config())
        self.context = context

        # Load dataset and get metadata.
        # This needs to be done before we initialize the HF config, tokenizer, and model
        # because we need to know num_labels before doing so.

        # For CSV/JSON files, this example will use as labels the column called `label` and as pair
        # of sentences the sentences in columns called `sentence1` and `sentence2` if such column
        # exists or the first two columns not named label if at least two columns are provided.
        #
        # If the CSVs/JSONs contain only one non-label column, the example will do single sentence
        # classification on this single column.

        # See more about loading any type of standard or custom dataset at
        # https://huggingface.co/docs/datasets/loading_datasets.html.

        self.raw_datasets = hf.default_load_dataset(self.data_config)

        if self.hparams.finetuning_task is not None:
            is_regression = self.hparams.finetuning_task == "stsb"
            if not is_regression:
                label_list = self.raw_datasets["train"].features["label"].names
                num_labels = len(label_list)
            else:
                num_labels = 1
        else:
            # Trying to have good defaults here, don't hesitate to tweak to your needs.
            is_regression = self.raw_datasets["train"].features["label"].dtype in [
                "float32",
                "float64",
            ]
            if is_regression:
                num_labels = 1
            else:
                # A useful fast method is datasets.Dataset.unique from
                # https://huggingface.co/docs/datasets/package_reference/main_classes.html
                label_list = self.raw_datasets["train"].unique("label")
                label_list.sort()  # Let's sort it for determinism
                num_labels = len(label_list)
        self.is_regression = is_regression
        self.hparams.num_labels = num_labels
        if not self.is_regression:
            self.label_list = label_list

        super(GLUETrial, self).__init__(context)
        self.logger.info(self.config)

        # We need to create the tokenized dataset after init because we need to model and
        # tokenizer to be available.
        self.tokenized_datasets = self.build_datasets()
        train_length = len(self.tokenized_datasets["train"])
        self.logger.info("training records: {}".format(train_length))
        if (
            "records_per_epoch" in self.exp_config
            and train_length != self.exp_config["records_per_epoch"]
        ):
            self.logger.warning(
                "number of train records {} does not match records_per_epoch of {}".format(
                    train_length, self.exp_config["records_per_epoch"]
                )
            )

        # Create metric reducer
        metric = datasets.load_metric("glue", self.hparams.finetuning_task)

        # You can define your custom compute_metrics function. It takes an `EvalPrediction` object
        # (a namedtuple with a predictions and label_ids field) and has to return a dictionary
        # mapping string to float.
        def compute_metrics(pred_labels) -> Dict:
            preds, labels = zip(*pred_labels)
            preds = utils.expand_like(preds)
            labels = utils.expand_like(labels)
            preds = np.squeeze(preds) if is_regression else np.argmax(preds, axis=1)
            if self.hparams.finetuning_task is not None:
                result = metric.compute(predictions=preds, references=labels)
                if len(result) > 1:
                    result["combined_score"] = np.mean(list(result.values())).item()
                return result
            elif is_regression:
                return {"mse": ((preds - labels) ** 2).mean().item()}
            else:
                return {"accuracy": (preds == labels).astype(np.float32).mean().item()}

        self.reducer = context.experimental.wrap_reducer(compute_metrics, for_training=False)

    def build_datasets(self) -> Union[datasets.Dataset, datasets.DatasetDict]:
        # Preprocessing the datasets
        if self.hparams.finetuning_task is not None:
            sentence1_key, sentence2_key = task_to_keys[self.hparams.finetuning_task]
        else:
            # We try to have some nice defaults but don't hesitate to tweak to your use case.
            non_label_column_names = [
                name for name in self.raw_datasets["train"].column_names if name != "label"
            ]
            if "sentence1" in non_label_column_names and "sentence2" in non_label_column_names:
                sentence1_key, sentence2_key = "sentence1", "sentence2"
            else:
                if len(non_label_column_names) >= 2:
                    sentence1_key, sentence2_key = non_label_column_names[:2]
                else:
                    sentence1_key, sentence2_key = non_label_column_names[0], None

        # Padding strategy
        if self.data_config.pad_to_max_length:
            padding = "max_length"
        else:
            # We will pad later, dynamically at batch creation to the max_seq_length in each batch.
            padding = False

        # Some models have set the order of the labels to use, so let's make sure we do use it.
        label_to_id = None
        if (
            self.model.config.label2id
            != transformers.PretrainedConfig(num_labels=self.hparams.num_labels).label2id
            and self.hparams.finetuning_task is not None
            and not self.is_regression
        ):
            # Some have all caps in their config, some don't.
            label_name_to_id = {k.lower(): v for k, v in self.model.config.label2id.items()}
            if sorted(label_name_to_id.keys()) == sorted(self.label_list):
                label_to_id = {
                    i: label_name_to_id[self.label_list[i]] for i in range(self.hparams.num_labels)
                }
            else:
                self.logger.warning(
                    "Your model seems to have been trained with labels, but they don't match the "
                    f"dataset: model labels: {sorted(label_name_to_id.keys())}, "
                    f"dataset labels: {sorted(self.label_list)}."
                    "\nIgnoring the model labels as a result.",
                )
        elif self.hparams.finetuning_task is None and not self.is_regression:
            label_to_id = {v: i for i, v in enumerate(self.label_list)}

        if self.data_config.max_seq_length > self.tokenizer.model_max_length:
            self.logger.warning(
                f"The max_seq_length passed ({self.data_config.max_seq_length}) is larger than "
                f"the maximum length for the model ({self.tokenizer.model_max_length}). Using "
                f"max_seq_length={self.tokenizer.model_max_length}."
            )
        max_seq_length = min(self.data_config.max_seq_length, self.tokenizer.model_max_length)

        # We cannot use self.tokenizer as a non-local variable in the preprocess_function if we
        # want map to be able to cache the output of the tokenizer.  Hence, the preprocess_function
        # takes a tokenizer explicitly as an input and we create a closure using functools.partial.
        def preprocess_function(tokenizer, padding, max_seq_length, examples):
            # Tokenize the texts
            args = (
                (examples[sentence1_key],)
                if sentence2_key is None
                else (examples[sentence1_key], examples[sentence2_key])
            )
            result = tokenizer(*args, padding=padding, max_length=max_seq_length, truncation=True)

            # Map labels to IDs (not necessary for GLUE tasks)
            if label_to_id is not None and "label" in examples:
                result["label"] = [label_to_id[label] for label in examples["label"]]
            return result

        tokenized_datasets = self.raw_datasets.map(
            functools.partial(preprocess_function, self.tokenizer, padding, max_seq_length),
            batched=True,
            load_from_cache_file=not self.data_config.overwrite_cache,
        )
        for _, data in tokenized_datasets.items():
            hf.remove_unused_columns(self.model, data)

        # Data collator will default to DataCollatorWithPadding, so we change it if we already
        # did the padding.
        if self.data_config.pad_to_max_length:
            self.collator = transformers.default_data_collator
        elif self.hparams.use_apex_amp:
            collator = transformers.DataCollatorWithPadding(self.tokenizer, pad_to_multiple_of=8)
            self.collator = lambda x: collator(x).data
        else:
            self.collator = None
        return tokenized_datasets

    def build_training_data_loader(self) -> det_torch.DataLoader:
        return det_torch.DataLoader(
            self.tokenized_datasets["train"],
            batch_size=self.context.get_per_slot_batch_size(),
            collate_fn=self.collator,
        )

    def build_validation_data_loader(self) -> det_torch.DataLoader:
        eval_dataset = self.tokenized_datasets[
            "validation_matched" if self.hparams.finetuning_task == "mnli" else "validation"
        ]
        return det_torch.DataLoader(
            eval_dataset,
            batch_size=self.context.get_per_slot_batch_size(),
            collate_fn=self.collator,
        )

    def evaluate_batch(self, batch: det_torch.TorchData, batch_idx: int) -> Dict:
        outputs = self.model(**batch)
        tmp_eval_loss, logits = outputs[:2]
        preds = logits.detach().cpu().numpy()
        out_label_ids = batch["labels"].detach().cpu().numpy()
        self.reducer.update((preds, out_label_ids))
        # We will return just the metrics outputed by the reducer.
        return {}
