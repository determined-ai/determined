"""
This example is largely based on the multiple-choice example in huggingface transformers for
the SWAG dataset. The license for the transformer's library is reproduced below.

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

import datasets
import numpy as np
import transformers
from data import DataCollatorForMultipleChoice

import determined.pytorch as det_torch
import model_hub.huggingface as hf


class SWAGTrial(hf.BaseTransformerTrial):
    def __init__(self, context: det_torch.PyTorchTrialContext) -> None:
        self.logger = logging.getLogger(__name__)
        super(SWAGTrial, self).__init__(context)
        self.logger.info(self.config)

        # Prep dataset
        # Get the datasets: you can either provide your own CSV or JSON training and evaluation
        # files (see below) or just provide the name of one of the public datasets available on the
        # hub at https://huggingface.co/datasets/ (the dataset will be downloaded automatically
        # from the datasets Hub).

        # For CSV/JSON files, this script will use the column called 'text' or the first column if
        # no column called 'text' is found. You can easily tweak this behavior (see below).

        # See more about loading any type of standard or custom dataset (from files, python dict,
        # pandas DataFrame, etc) at
        # https://huggingface.co/docs/datasets/loading_datasets.html.
        self.raw_datasets = hf.default_load_dataset(self.data_config)
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

    def build_datasets(self) -> Union[datasets.Dataset, datasets.DatasetDict]:
        # When using your own dataset or a different dataset from swag, you will probably need
        # to change this.
        ending_names = [f"ending{i}" for i in range(4)]
        context_name = "sent1"
        question_header_name = "sent2"

        padding = "max_length" if self.data_config.pad_to_max_length else False
        if self.data_config.max_seq_length is None:
            max_seq_length = self.tokenizer.model_max_length
            if max_seq_length > 1024:
                self.logger.warning(
                    "The tokenizer picked seems to have a very large `model_max_length` "
                    f"({self.tokenizer.model_max_length}). Using 1024 instead. You can change "
                    "that default value by setting max_seq_length in the experiment config."
                )
                max_seq_length = 1024
        else:
            if self.data_config.max_seq_length > self.tokenizer.model_max_length:
                self.logger.warning(
                    f"The max_seq_length passed ({self.data_config.max_seq_length}) is larger "
                    f"than the maximum length for the model ({self.tokenizer.model_max_length}). "
                    f"Using max_seq_length={self.tokenizer.model_max_length}."
                )
            max_seq_length = min(self.data_config.max_seq_length, self.tokenizer.model_max_length)

        # We cannot use self.tokenizer as a non-local variable in the preprocess_function if we
        # want map to be able to cache the output of the tokenizer.  Hence, the preprocess_function
        # takes a tokenizer explicitly as an input and we create a closure using functools.partial.
        def preprocess_function(tokenizer, padding, max_seq_length, examples):
            first_sentences = [[context] * 4 for context in examples[context_name]]
            question_headers = examples[question_header_name]
            second_sentences = [
                [f"{header} {examples[end][i]}" for end in ending_names]
                for i, header in enumerate(question_headers)
            ]

            # Flatten out
            first_sentences = sum(first_sentences, [])
            second_sentences = sum(second_sentences, [])

            # Tokenize
            tokenized_examples = tokenizer(
                first_sentences,
                second_sentences,
                truncation=True,
                max_length=max_seq_length,
                padding=padding,
            )
            # Un-flatten
            return {
                k: [v[i : i + 4] for i in range(0, len(v), 4)]
                for k, v in tokenized_examples.items()
            }

        tokenized_datasets = self.raw_datasets.map(
            functools.partial(preprocess_function, self.tokenizer, padding, max_seq_length),
            batched=True,
            num_proc=self.data_config.preprocessing_num_workers,
            load_from_cache_file=not self.data_config.overwrite_cache,
        )
        for _, data in tokenized_datasets.items():
            hf.remove_unused_columns(self.model, data)

        # Data collator
        self.collator = (
            transformers.default_data_collator
            if self.data_config.pad_to_max_length
            else DataCollatorForMultipleChoice(tokenizer=self.tokenizer)
        )
        return tokenized_datasets

    def build_training_data_loader(self) -> det_torch.DataLoader:
        return det_torch.DataLoader(
            self.tokenized_datasets["train"],
            batch_size=self.context.get_per_slot_batch_size(),
            collate_fn=self.collator,
        )

    def build_validation_data_loader(self) -> det_torch.DataLoader:
        return det_torch.DataLoader(
            self.tokenized_datasets["validation"],
            batch_size=self.context.get_per_slot_batch_size(),
            collate_fn=self.collator,
        )

    def evaluate_batch(self, batch: det_torch.TorchData, batch_idx: int) -> Dict:
        outputs = self.model(**batch)
        tmp_eval_loss, logits = outputs[:2]
        preds = logits.detach().cpu().numpy()
        preds = np.argmax(preds, axis=1)
        label_ids = batch["labels"].detach().cpu().numpy()
        return {"accuracy": (preds == label_ids).astype(np.float32).mean().item()}
