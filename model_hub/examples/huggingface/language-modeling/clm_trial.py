"""
This example is largely based on the causal language modeling example in huggingface transformers.
The license for the transformer's library is reproduced below.

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

import determined.pytorch as det_torch
import model_hub.huggingface as hf


class CLMTrial(hf.BaseTransformerTrial):
    def __init__(self, context: det_torch.PyTorchTrialContext) -> None:
        self.logger = logging.getLogger(__name__)
        super(CLMTrial, self).__init__(context)
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
        self.model.resize_token_embeddings(len(self.tokenizer))
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

        self.reducer = self.context.experimental.wrap_reducer(
            lambda losses: np.exp(np.mean(losses)), name="perplexity", for_training=False
        )

    def build_datasets(self) -> Union[datasets.Dataset, datasets.DatasetDict]:
        column_names = self.raw_datasets["train"].column_names
        text_column_name = "text" if "text" in column_names else column_names[0]

        def tokenize_function(tokenizer, examples):
            return tokenizer(examples[text_column_name])

        # We cannot use self.tokenizer as a non-local variable in the tokenize_function if we want
        # map to be able to cache the output of the tokenizer.  Hence, the tokenize_function takes
        # a tokenizer explicitly as an input and we create a closure using functools.partial.
        tokenized_datasets = self.raw_datasets.map(
            functools.partial(tokenize_function, self.tokenizer),
            batched=True,
            num_proc=self.data_config.preprocessing_num_workers,
            remove_columns=column_names,
            load_from_cache_file=not self.data_config.overwrite_cache,
        )

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

        # Main data processing function that will concatenate all texts from our dataset and
        # generate chunks of max_seq_length.
        def group_texts(examples):
            # Concatenate all texts.
            concatenated_examples = {k: sum(examples[k], []) for k in examples.keys()}
            total_length = len(concatenated_examples[list(examples.keys())[0]])
            # We drop the small remainder, we could add padding if the model supported it instead
            # of this drop, you can customize this part to your needs.
            total_length = (total_length // max_seq_length) * max_seq_length
            # Split by chunks of max_len.
            result = {
                k: [t[i : i + max_seq_length] for i in range(0, total_length, max_seq_length)]
                for k, t in concatenated_examples.items()
            }
            result["labels"] = result["input_ids"].copy()
            return result

        # Note that with `batched=True`, this map processes 1,000 texts together, so
        # group_texts throws away a remainder for each of those groups of 1,000 texts.
        # You can adjust that batch_size here but a higher value might be slower to preprocess.
        #
        # To speed up this part, we use multiprocessing. See the documentation of the map
        # method for more information:
        # https://huggingface.co/docs/datasets/package_reference/main_classes.html
        lm_datasets = tokenized_datasets.map(
            group_texts,
            batched=True,
            num_proc=self.data_config.preprocessing_num_workers,
            load_from_cache_file=not self.data_config.overwrite_cache,
        )
        for _, data in tokenized_datasets.items():
            hf.remove_unused_columns(self.model, data)

        self.collator = transformers.default_data_collator

        return lm_datasets

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
        self.reducer.update(outputs[0].detach().cpu().numpy())
        # Although we are returning the empty dictionary below, we will still get the metrics from
        # custom reducer that we passed to the context during initialization.
        return {}
