import functools
import logging

import ner_utils
import torch
import transformers
from attrdict import AttrDict
from model_hub import huggingface as hf
from model_hub import utils

from determined.pytorch import DataLoader, LRScheduler, PyTorchTrialContext


class NERTrial(hf.BaseTransformerTrial):
    def __init__(self, context: PyTorchTrialContext) -> None:
        utils.configure_logging(
            logging.INFO if context.distributed.get_local_rank() in [-1, 0] else logging.WARN
        )
        self.logger = logging.getLogger(__name__)
        self.hparams = AttrDict(context.get_hparams())
        self.data_config = AttrDict(context.get_data_config())
        self.context = context

        # Prep dataset
        self.raw_datasets = hf.default_load_dataset(self.data_config)
        datasets_metadata = ner_utils.get_dataset_metadata(self.raw_datasets, self.hparams)
        self.hparams.num_labels = datasets_metadata.num_labels

        # Parse hparams and init model, optimizer, and lr_scheduler.
        # See model_hub/huggingface/_arg_parser.py for default expected args.
        config_args, tokenizer_args, model_args = hf.default_parse_config_tokenizer_model_args(
            self.hparams
        )
        optimizer_args, scheduler_args = hf.default_parse_optimizer_lr_scheduler_args(self.hparams)
        self.logger.info("Config arg:")
        self.logger.info(config_args)
        self.logger.info("Tokenizer arg:")
        self.logger.info(tokenizer_args)
        self.logger.info("Model arg:")
        self.logger.info(model_args)
        self.logger.info("Optimizer arg:")
        self.logger.info(optimizer_args)
        self.logger.info("LR Scheduler arg:")
        self.logger.info(scheduler_args)

        # This is used in the backward step called in train_batch of BaseTransformerTrial.
        self.grad_clip_fn = (
            lambda x: torch.nn.utils.clip_grad_norm_(x, optimizer_args.max_grad_norm)
            if optimizer_args.max_grad_norm > 0
            else None
        )

        self.config, self.tokenizer, self.model = hf.build_using_auto(
            config_args, tokenizer_args, self.hparams.model_mode, model_args
        )
        self.model = self.context.wrap_model(self.model)

        # The map function for datasets requires all objects be pickle-able for now.  This means
        # we need to call this function before we create the optimizer, which is not pickle-able.
        # Once this requirement is removed from huggingface datasets, we can simply call
        # super(NERTrial, self).__init__(context) before creating the tokenized datasets.
        # See https://github.com/huggingface/datasets/pull/1703.
        self.tokenized_datasets = ner_utils.build_tokenized_datasets(
            self.raw_datasets,
            self.model,
            self.data_config,
            self.tokenizer,
            datasets_metadata.text_column_name,
            datasets_metadata.label_column_name,
            datasets_metadata.label_to_id,
        )

        self.optimizer = self.context.wrap_optimizer(
            hf.build_default_optimizer(self.model, optimizer_args)
        )

        if self.hparams.use_apex_amp:
            self.model, self.optimizer = self.context.configure_apex_amp(
                models=self.model,
                optimizers=self.optimizer,
            )

        self.lr_scheduler = self.context.wrap_lr_scheduler(
            hf.build_default_lr_scheduler(self.optimizer, scheduler_args),
            LRScheduler.StepMode.STEP_EVERY_BATCH,
        )

        # Create metric reducer
        self.reducer = context.experimental.wrap_reducer(
            utils.PredLabelFnReducer(
                functools.partial(ner_utils.compute_metrics, datasets_metadata.label_list)
            ),
            for_training=False,
        )

    def build_training_data_loader(self) -> DataLoader:
        return DataLoader(
            self.tokenized_datasets["train"],
            batch_size=self.context.get_per_slot_batch_size(),
            collate_fn=transformers.DataCollatorForTokenClassification(self.tokenizer),
        )

    def build_validation_data_loader(self) -> DataLoader:
        return DataLoader(
            self.tokenized_datasets["validation"],
            batch_size=self.context.get_per_slot_batch_size(),
            collate_fn=transformers.DataCollatorForTokenClassification(self.tokenizer),
        )

    def evaluate_batch(self, batch):
        outputs = self.model(**batch)
        tmp_eval_loss, logits = outputs[:2]
        preds = logits.detach().cpu().numpy()
        out_label_ids = batch["labels"].detach().cpu().numpy()
        self.reducer.update(preds, out_label_ids)
        # We will return just the metrics outputed by the reducer.
        return {}
