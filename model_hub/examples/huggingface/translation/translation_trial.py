"""
This example is largely based on the translation example in the huggingface
transformers library. The license for the transformer's library is reproduced below.

==================================================================================================

Copyright The HuggingFace Team and The HuggingFace Inc. team. All rights reserved.

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
from typing import Any, Dict, Tuple, Union

import attrdict
import datasets
import numpy as np
import torch
import transformers

import determined.pytorch as det_torch
import model_hub.huggingface as hf
import model_hub.utils as utils

# A list of all multilingual tokenizer which require src_lang and tgt_lang attributes.
MULTILINGUAL_TOKENIZERS = [
    transformers.MBartTokenizer,
    transformers.MBartTokenizerFast,
    transformers.MBart50Tokenizer,
    transformers.MBart50TokenizerFast,
    transformers.M2M100Tokenizer,
]


class TranslationTrial(hf.BaseTransformerTrial):
    def __init__(self, context: det_torch.PyTorchTrialContext) -> None:
        self.logger = logging.getLogger(__name__)
        self.hparams = attrdict.AttrDict(context.get_hparams())
        self.data_config = attrdict.AttrDict(context.get_data_config())

        if (
            self.data_config.source_prefix is None
            and self.hparams.pretrained_model_name_or_path
            in [
                "t5-small",
                "t5-base",
                "t5-large",
                "t5-3b",
                "t5-11b",
            ]
        ):
            self.logger.warning(
                "You're running a t5 model but didn't provide a source prefix, which is expected, "
                "e.g. with `data.source_prefix 'translate English to German: ' `"
            )
        # Creates HF config, tokenizer, model using parent init function.
        super(TranslationTrial, self).__init__(context)
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
        if self.data_config.val_max_target_length is None:
            self.data_config.val_max_target_length = min(
                self.data_config.max_target_length, self.tokenizer.model_max_length
            )
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

        # Create metric reducer
        self.metric = datasets.load_metric("sacrebleu")
        self.reducer = context.experimental.wrap_reducer(
            functools.partial(self.compute_metrics, self.metric),
            for_training=False,
        )

    def compute_metrics(self, metric, eval_preds):
        preds, labels = zip(*eval_preds)
        if isinstance(preds[0], tuple):
            preds = [p[0] for p in preds]
        preds = utils.expand_like(preds)
        labels = utils.expand_like(labels)
        decoded_preds = self.tokenizer.batch_decode(preds, skip_special_tokens=True)
        if self.data_config.ignore_pad_token_for_loss:
            # Replace -100 in the labels as we can't decode them.
            cleaned_labels = np.where(labels != -100, labels, self.tokenizer.pad_token_id)
        decoded_labels = self.tokenizer.batch_decode(cleaned_labels, skip_special_tokens=True)

        # Some simple post-processing
        decoded_preds = [pred.strip() for pred in decoded_preds]
        decoded_labels = [[label.strip()] for label in decoded_labels]

        result = metric.compute(predictions=decoded_preds, references=decoded_labels)
        result = {"bleu": result["score"]}

        prediction_lens = [np.count_nonzero(pred != self.tokenizer.pad_token_id) for pred in preds]
        result["gen_len"] = np.mean(prediction_lens)
        result = {k: round(v, 4) for k, v in result.items()}
        return result

    def build_datasets(self) -> Dict[str, Union[datasets.Dataset, datasets.DatasetDict]]:
        self.model.resize_token_embeddings(len(self.tokenizer))

        # Set decoder_start_token_id
        if self.model.config.decoder_start_token_id is None and isinstance(
            self.tokenizer, (transformers.MBartTokenizer, transformers.MBartTokenizerFast)
        ):
            if isinstance(self.tokenizer, transformers.MBartTokenizer):
                self.model.config.decoder_start_token_id = self.tokenizer.lang_code_to_id[
                    self.data_config.target_lang
                ]
            else:
                self.model.config.decoder_start_token_id = self.tokenizer.convert_tokens_to_ids(
                    self.data_config.target_lang
                )

        if self.model.config.decoder_start_token_id is None:
            raise ValueError("Make sure that `config.decoder_start_token_id` is correctly defined")

        prefix = (
            self.data_config.source_prefix if self.data_config.source_prefix is not None else ""
        )
        column_names = self.raw_datasets["train"].column_names

        # For translation we set the codes of our source and target languages (only useful for
        # mBART, the others will ignore those attributes).
        if isinstance(self.tokenizer, tuple(MULTILINGUAL_TOKENIZERS)):
            assert (
                self.data_config.target_lang is not None
                and self.data_config.source_lang is not None
            ), (
                f"{self.tokenizer.__class__.__name__} is a multilingual tokenizer which requires "
                "data.source_lang and data.target_lang fields to be set."
            )

            self.tokenizer.src_lang = self.data_config.source_lang
            self.tokenizer.tgt_lang = self.data_config.target_lang

            # For multilingual translation models like mBART-50 and M2M100 we need to force the
            # target language token as the first generated token. We ask the user to explicitly
            # provide this via data.forced_bos_token field.
            forced_bos_token_id = (
                self.tokenizer.lang_code_to_id[self.data_config.forced_bos_token]
                if self.data_config.forced_bos_token is not None
                else None
            )
            self.model.config.forced_bos_token_id = forced_bos_token_id

        # Get the language codes for input/target.
        source_lang = self.data_config.source_lang.split("_")[0]
        target_lang = self.data_config.target_lang.split("_")[0]

        # Temporarily set max_target_length for training.
        max_target_length = min(self.data_config.max_target_length, self.tokenizer.model_max_length)
        padding = "max_length" if self.data_config.pad_to_max_length else False

        def preprocess_function(
            tokenizer,
            source_lang,
            target_lang,
            prefix,
            data_config,
            padding,
            max_target_length,
            examples,
        ):
            inputs = [ex[source_lang] for ex in examples["translation"]]
            targets = [ex[target_lang] for ex in examples["translation"]]
            inputs = [prefix + inp for inp in inputs]
            model_inputs = tokenizer(
                inputs, max_length=data_config.max_source_length, padding=padding, truncation=True
            )

            # Setup the tokenizer for targets
            with tokenizer.as_target_tokenizer():
                labels = tokenizer(
                    targets, max_length=max_target_length, padding=padding, truncation=True
                )

            # If we are padding here, replace all tokenizer.pad_token_id in the labels by -100
            # when we want to ignore padding in the loss.
            if padding == "max_length" and data_config.ignore_pad_token_for_loss:
                labels["input_ids"] = [
                    [(lb if lb != tokenizer.pad_token_id else -100) for lb in label]
                    for label in labels["input_ids"]
                ]

            model_inputs["labels"] = labels["input_ids"]
            return model_inputs

        tokenized_datasets = {}

        for split in ["train", "validation"]:
            max_target_length = (
                self.data_config.max_target_length
                if split == "train"
                else self.data_config.val_max_target_length
            )
            dataset = self.raw_datasets[split]
            tokenized_datasets[split] = dataset.map(
                functools.partial(
                    preprocess_function,
                    self.tokenizer,
                    source_lang,
                    target_lang,
                    prefix,
                    self.data_config,
                    padding,
                    max_target_length,
                ),
                batched=True,
                num_proc=self.data_config.preprocessing_num_workers,
                remove_columns=column_names,
                load_from_cache_file=not self.data_config.overwrite_cache,
            )
            hf.remove_unused_columns(self.model, tokenized_datasets[split])

        # Data collator
        label_pad_token_id = (
            -100 if self.data_config.ignore_pad_token_for_loss else self.tokenizer.pad_token_id
        )
        if self.data_config.pad_to_max_length:
            collator = transformers.default_data_collator
        else:
            collator = transformers.DataCollatorForSeq2Seq(
                self.tokenizer,
                model=self.model,
                label_pad_token_id=label_pad_token_id,
                pad_to_multiple_of=8 if self.hparams.use_apex_amp else None,
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

    def pad_tensors_to_max_len(self, tensor, max_length):
        if self.tokenizer is None:
            raise ValueError(
                f"Tensor need to be padded to `max_length={max_length}` but no tokenizer was "
                "passed when creating this `Trainer`. Make sure to create your `Trainer` with "
                "the appropriate tokenizer."
            )
        # If PAD token is not defined at least EOS token has to be defined
        pad_token_id = (
            self.tokenizer.pad_token_id
            if self.tokenizer.pad_token_id is not None
            else self.tokenizer.eos_token_id
        )

        padded_tensor = pad_token_id * torch.ones(
            (tensor.shape[0], max_length), dtype=tensor.dtype, device=tensor.device
        )
        padded_tensor[:, : tensor.shape[-1]] = tensor
        return padded_tensor

    def prediction_step(
        self,
        inputs: Dict[str, Union[torch.Tensor, Any]],
    ) -> Tuple[float, torch.Tensor, torch.Tensor]:
        """
        Perform an evaluation step on :obj:`model` using obj:`inputs`.
        Args:
            inputs (:obj:`Dict[str, Union[torch.Tensor, Any]]`):
                The inputs and targets of the model.
                The dictionary will be unpacked before being fed to the model. Most models expect
                the targets under the argument :obj:`labels`. Check your model's documentation
                for all accepted arguments.
        Return:
            Tuple[float, torch.Tensor, torch.Tensor]: A tuple with the loss, logits and
            labels.
        """
        gen_kwargs = {
            "max_length": self.data_config.val_max_target_length,
            "num_beams": self.data_config.num_beams,
            "synced_gpus": False,
        }

        generated_tokens = self.model.generate(
            inputs["input_ids"],
            attention_mask=inputs["attention_mask"],
            **gen_kwargs,
        )
        # in case the batch is shorter than max length, the output should be padded
        if generated_tokens.shape[-1] < gen_kwargs["max_length"]:
            generated_tokens = self.pad_tensors_to_max_len(
                generated_tokens, gen_kwargs["max_length"]
            )

        with torch.no_grad():
            outputs = self.model(**inputs)
            loss = (outputs["loss"] if isinstance(outputs, dict) else outputs[0]).mean().detach()

        labels = inputs["labels"]
        if labels.shape[-1] < gen_kwargs["max_length"]:
            labels = self.pad_tensors_to_max_len(labels, gen_kwargs["max_length"])

        return (loss, generated_tokens, labels)

    def evaluate_batch(self, batch: det_torch.TorchData, batch_idx: int) -> Dict:
        loss, logits, labels = self.prediction_step(batch)
        preds = logits.detach().cpu().numpy()
        out_label_ids = labels.detach().cpu().numpy()  # type: ignore
        self.reducer.update((preds, out_label_ids))  # type: ignore
        return {"loss": loss}
