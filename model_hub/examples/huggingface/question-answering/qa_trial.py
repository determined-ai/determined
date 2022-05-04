"""
This example is largely based on the question-answering example in the huggingface
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

import data
import datasets
import qa_utils
import transformers

import determined.pytorch as det_torch
import model_hub.huggingface as hf


class QATrial(hf.BaseTransformerTrial):
    def __init__(self, context: det_torch.PyTorchTrialContext) -> None:
        self.logger = logging.getLogger(__name__)
        super(QATrial, self).__init__(context)
        self.logger.info(self.config)

        # Check to make sure the dataset is configured correctly.
        if self.data_config.dataset_name is not None:
            dataset_name = self.data_config.dataset_name
            if dataset_name == "squad":
                assert (
                    not self.data_config.version_2_with_negative
                ), "version_2_with_negative should be false for squad"
            elif dataset_name == "squad_v2":
                assert (
                    self.data_config.version_2_with_negative
                ), "version_2_with_negative should be true for squad_v2"

        self.data_processors = data

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
        self.column_names = self.raw_datasets["train"].column_names

        if not isinstance(self.tokenizer, transformers.PreTrainedTokenizerFast):
            raise ValueError(
                "This example script only works for models that have a fast tokenizer. Checkout "
                "the big table of models at "
                "https://huggingface.co/transformers/index.html#bigtable to find the model types "
                "that meet this requirement"
            )

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
        metric = datasets.load_metric(
            "squad_v2" if self.data_config.version_2_with_negative else "squad"
        )

        self.reducer = context.wrap_reducer(
            functools.partial(
                qa_utils.compute_metrics,
                self.data_config,
                self.column_names,
                self.data_processors.post_processing_function,
                self.raw_datasets,
                self.tokenized_datasets,
                self.model,
                metric,
            ),
            for_training=False,
        )

    def build_datasets(self) -> Dict[str, Union[datasets.Dataset, datasets.DatasetDict]]:
        tokenized_datasets = {}
        for split in ["train", "validation"]:
            tokenized_datasets[split] = self.raw_datasets[split].map(
                functools.partial(
                    self.data_processors.prepare_features,
                    split,
                    self.data_config,
                    self.tokenizer,
                    self.column_names,
                ),
                batched=True,
                num_proc=self.data_config.preprocessing_num_workers,
                remove_columns=self.column_names,
                load_from_cache_file=not self.data_config.overwrite_cache,
            )
            hf.remove_unused_columns(self.model, tokenized_datasets[split])
        if self.data_config.pad_to_max_length:
            self.collator = transformers.default_data_collator
        else:
            collator = transformers.DataCollatorWithPadding(
                self.tokenizer, pad_to_multiple_of=8 if self.hparams.use_apex_amp else None
            )
            self.collator = lambda x: collator(x).data
        return tokenized_datasets

    def build_training_data_loader(self) -> det_torch.DataLoader:
        return det_torch.DataLoader(
            self.tokenized_datasets["train"],
            batch_size=self.context.get_per_slot_batch_size(),
            collate_fn=self.collator,
        )

    def build_validation_data_loader(self) -> det_torch.DataLoader:
        # Determined's distributed batch sampler interleaves shards on each GPU slot so
        # sample i goes to worker with rank i % world_size.  Therefore, we need to re-sort
        # all the samples once we gather the predictions before computing the validation metric.
        return det_torch.DataLoader(
            qa_utils.DatasetWithIndex(self.tokenized_datasets["validation"]),
            batch_size=self.context.get_per_slot_batch_size(),
            collate_fn=self.collator,
        )

    def evaluate_batch(self, batch: det_torch.TorchData, batch_idx: int) -> Dict:
        ind = batch.pop("ind")
        outputs = self.model(**batch)
        if isinstance(outputs, dict):
            predictions = tuple(
                v.detach().cpu().numpy() for k, v in outputs.items() if k not in ("loss", "mems")
            )
        else:
            predictions = outputs[1:].detach().cpu().numpy()

        self.reducer.update((ind.detach().cpu().numpy(), predictions))
        # Although we are returning the empty dictionary below, we will still get the metrics from
        # custom reducer that we passed to the context during initialization.
        return {}
