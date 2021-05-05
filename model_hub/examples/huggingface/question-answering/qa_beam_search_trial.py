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

import attrdict
import data_beam_search
import datasets
import torch
import transformers

import determined.pytorch as det_torch
import model_hub.huggingface as hf
import model_hub.utils as utils


class QABeamSearchTrial(hf.BaseTransformerTrial):
    def __init__(self, context: det_torch.PyTorchTrialContext) -> None:
        self.logger = logging.getLogger(__name__)
        self.hparams = attrdict.AttrDict(context.get_hparams())
        self.data_config = attrdict.AttrDict(context.get_data_config())
        self.context = context

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

        self.data_processors = data_beam_search

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

        # For beam search, we need to use a different model from the default model returned by
        # AutoModelForQuestionAnswering.  We will use a custom init in this case that is a slight
        # modification of the BaseTransformerTrial init method.
        self.exp_config = attrdict.AttrDict(context.get_experiment_config())

        # Check to make sure all expected hyperparameters are set.
        self.check_hparams()

        # Parse hparams and data_config.
        (
            self.config_kwargs,
            self.tokenizer_kwargs,
            self.model_kwargs,
        ) = hf.default_parse_config_tokenizer_model_kwargs(self.hparams)
        optimizer_kwargs, scheduler_kwargs = hf.default_parse_optimizer_lr_scheduler_kwargs(
            self.hparams
        )

        self.config = transformers.XLNetConfig.from_pretrained(**self.config_kwargs)
        self.tokenizer = transformers.XLNetTokenizerFast.from_pretrained(**self.tokenizer_kwargs)

        # We need to use XLNetForQuestionAnswering instead of XLNetForQuestionAnsweringSimple
        # which is the default returned by AutoModelForQuestionAnswering.
        if self.hparams.use_pretrained_weights:
            self.model_kwargs["config"] = self.config
            self.model = transformers.XLNetForQuestionAnswering.from_pretrained(**self.model_kwargs)
        else:
            self.model = transformers.XLNetForQuestionAnswering(self.config)
        self.model = self.context.wrap_model(self.model)

        # The rest is the same as the parent init method.
        self.optimizer = self.context.wrap_optimizer(
            hf.build_default_optimizer(self.model, optimizer_kwargs)
        )

        if self.hparams.use_apex_amp:
            self.model, self.optimizer = self.context.configure_apex_amp(
                models=self.model,
                optimizers=self.optimizer,
            )

        self.lr_scheduler = self.context.wrap_lr_scheduler(
            hf.build_default_lr_scheduler(self.optimizer, scheduler_kwargs),
            det_torch.LRScheduler.StepMode.STEP_EVERY_BATCH,
        )
        self.grad_clip_fn = (
            lambda x: torch.nn.utils.clip_grad_norm_(x, optimizer_kwargs.max_grad_norm)
            if optimizer_kwargs.max_grad_norm > 0  # type: ignore
            else None
        )

        self.logger.info(self.config)

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

        def compute_metrics(predictions):
            predictions = zip(*predictions)
            predictions = [utils.expand_like(p) for p in predictions]
            # We need to add back in columns needed for validation.
            self.tokenized_datasets["validation"].set_format(
                type=self.tokenized_datasets["validation"].format["type"],
                columns=list(self.tokenized_datasets["validation"].features.keys()),
            )
            output = self.data_processors.post_processing_function(
                examples=self.raw_datasets["validation"],
                features=self.tokenized_datasets["validation"],
                predictions=predictions,
                data_args=self.data_config,
                column_names=self.column_names,
                prefix="eval",
                model=self.model,
            )
            result = metric.compute(predictions=output.predictions, references=output.label_ids)
            # Then remove them again so that data collation doesn't break.
            hf.remove_unused_columns(self.model, self.tokenized_datasets["validation"])
            return result

        self.reducer = context.experimental.wrap_reducer(compute_metrics, for_training=False)

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
        return det_torch.DataLoader(
            self.tokenized_datasets["validation"],
            batch_size=self.context.get_per_slot_batch_size(),
            collate_fn=self.collator,
        )

    def evaluate_batch(self, batch: det_torch.TorchData, batch_idx: int) -> Dict:
        outputs = self.model(**batch)
        if isinstance(outputs, dict):
            predictions = tuple(
                v.detach().cpu().numpy() for k, v in outputs.items() if k not in ("loss", "mems")
            )
        else:
            predictions = outputs[1:].detach().cpu().numpy()

        self.reducer.update(predictions)
        # Although we are returning the empty dictionary below, we will still get the metrics from
        # custom reducer that we passed to the context during initialization.
        return {}
