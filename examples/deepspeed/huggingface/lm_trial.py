import logging
from typing import Any, Dict, Iterator, Optional, Union

import datasets
import deepspeed
import model_hub.huggingface as hf
import torch
from attrdict import AttrDict
from determined.pytorch import DataLoader, TorchData
from determined.pytorch.deepspeed import DeepSpeedTrial, DeepSpeedTrialContext, overwrite_deepspeed_config
from transformers import default_data_collator


class HFDeepSpeedTrial(DeepSpeedTrial):
    def __init__(self, context: DeepSpeedTrialContext) -> None:
        self.logger = logging.getLogger(__name__)
        self.context = context
        self.hparams = AttrDict(self.context.get_hparams())
        self.data_config = AttrDict(self.context.get_data_config())
        self.exp_config = AttrDict(self.context.get_experiment_config())

        # Parse hparams and data_config.
        (
            self.config_kwargs,
            self.tokenizer_kwargs,
            self.model_kwargs,
        ) = hf._config_parser.default_parse_config_tokenizer_model_kwargs(self.hparams)

        (
            self.config,
            self.tokenizer,
            self.model
        ) = hf.build_using_auto(self.config_kwargs,
                                self.tokenizer_kwargs,
                                self.hparams.model_mode,
                                self.model_kwargs,
                                use_pretrained_weights=self.hparams.use_pretrained_weights)

        self.tokenized_datasets = datasets.load_from_disk(self.data_config.preprocessed_dataset_path)
        for _, data in self.tokenized_datasets.items():
            hf.remove_unused_columns(self.model, data)

        self.model.resize_token_embeddings(len(self.tokenizer))

        # Overwrite values in the deepspeed config json with values from the Determined context
        overwrite_deepspeed_args = self.hparams.get("overwrite_deepspeed_args", {})

        # Make sure the optimizer LR and max LR for the WarmupLR scheduler are the same
        try:
            optimizer_lr = overwrite_deepspeed_args["optimizer"]["params"]["lr"]
            overwrite_deepspeed_args["scheduler"] = {"params": {"warmup_max_lr": optimizer_lr}}
        except KeyError:
            # There was no Optimizer LR hyperparameter
            pass

        self.ds_config = overwrite_deepspeed_config(self.hparams.deepspeed_config_file,
                                                    overwrite_deepspeed_args)

        parameters = filter(lambda p: p.requires_grad, self.model.parameters())
        model_engine, _, _, _ = deepspeed.initialize(model=self.model,
                                                     model_parameters=parameters,
                                                     config=self.ds_config)
        self.model_engine = self.context.wrap_model_engine(model_engine)

    def build_training_data_loader(self) -> DataLoader:
        return DataLoader(
            self.tokenized_datasets["train"],
            batch_size=self.ds_config["train_micro_batch_size_per_gpu"],
            collate_fn=default_data_collator,
        )

    def build_validation_data_loader(self) -> DataLoader:
        return DataLoader(self.tokenized_datasets["validation"],
                          batch_size=self.ds_config["train_micro_batch_size_per_gpu"],
                          collate_fn=default_data_collator)

    def train_batch(self,
                    dataloader_iter: Optional[Iterator[TorchData]],
                    epoch_idx: int,
                    batch_idx: int) -> Union[torch.Tensor, Dict[str, Any]]:
        inputs = self.context.to_device(next(dataloader_iter))
        outputs = self.model_engine(**inputs)
        loss = outputs["loss"] if isinstance(outputs, dict) else outputs[0]
        self.model_engine.backward(loss)
        self.model_engine.step()
        return {"loss": loss}

    def evaluate_batch(self,
                       dataloader_iter: Optional[Iterator[TorchData]],
                       batch_idx: int) -> Dict[str, Any]:
        inputs = self.context.to_device(next(dataloader_iter))
        outputs = self.model_engine(**inputs)
        loss = outputs["loss"] if isinstance(outputs, dict) else outputs[0]
        return {"loss": loss}
