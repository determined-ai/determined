import dataclasses
import logging
from typing import Any, Dict, List, Optional, Tuple, Union, cast

import attrdict
import datasets as hf_datasets
import torch
import transformers
import transformers.optimization as hf_opt

import determined.pytorch as det_torch
import model_hub.utils
from determined.common.api.analytics import send_analytics
from model_hub.huggingface import _config_parser as hf_parse

MODEL_MODES = {
    "base": transformers.AutoModel,
    "pretraining": transformers.AutoModelForPreTraining,
    "causal-lm": transformers.AutoModelForCausalLM,
    "masked-lm": transformers.AutoModelForMaskedLM,
    "seq2seq-lm": transformers.AutoModelForSeq2SeqLM,
    "sequence-classification": transformers.AutoModelForSequenceClassification,
    "multiple-choice": transformers.AutoModelForMultipleChoice,
    "next-sentence": transformers.AutoModelForNextSentencePrediction,
    "token-classification": transformers.AutoModelForTokenClassification,
    "question-answering": transformers.AutoModelForQuestionAnswering,
}


def build_using_auto(
    config_kwargs: Union[Dict, attrdict.AttrDict],
    tokenizer_kwargs: Union[Dict, attrdict.AttrDict],
    model_mode: str,
    model_kwargs: Union[Dict, attrdict.AttrDict],
    use_pretrained_weights: bool = True,
) -> Tuple[
    transformers.PretrainedConfig,  # This is how it's named in transformers
    transformers.PreTrainedTokenizer,
    transformers.PreTrainedModel,
]:
    """
    Build the config, tokenizer, and model using tranformer's
    Auto classes.

    Args:
        config_kwargs: arguments for transformers configuration classes
        tokenizer_kwargs: arguments for transformers tokenizer classes
        model_mode: one of (pretraining, causal-lm, masked-lm, seq2seq-lm, sequence-classification,
            multiple-choice, next-sentence, token-classification, question-answering)
        model_kwargs: arguments for transformers model classes

    Returns:
        transformer config, tokenizer, and model
    """
    config = transformers.AutoConfig.from_pretrained(**config_kwargs)
    tokenizer = transformers.AutoTokenizer.from_pretrained(**tokenizer_kwargs)
    model_builder = MODEL_MODES[model_mode]
    if isinstance(model_kwargs, hf_parse.ModelKwargs):
        model_kwargs = dataclasses.asdict(model_kwargs)
    if use_pretrained_weights:
        model_kwargs["config"] = config
        model = model_builder.from_pretrained(**model_kwargs)
    else:
        model = model_builder.from_config(config)
    return config, tokenizer, model


def group_parameters_for_optimizer(
    model: torch.nn.Module,
    weight_decay: Optional[float] = 0,
    no_decay: Tuple[str, ...] = ("bias", "LayerNorm.weight"),
) -> List[Dict[str, Any]]:
    """
    Group parameters by whether weight_decay is applied or not.

    Args:
        model: model supplying the learnable parameters
        weight_decay: value for weight_decay
        no_decay: variable names that should not have weight_decay applied
    Returns:
        grouped parameters according to whether weight_decay should be applied
    """
    return [
        {
            "params": [
                p for n, p in model.named_parameters() if not any(nd in n for nd in no_decay)
            ],
            "weight_decay": weight_decay,
        },
        {
            "params": [p for n, p in model.named_parameters() if any(nd in n for nd in no_decay)],
            "weight_decay": 0.0,
        },
    ]


def build_default_optimizer(
    model: torch.nn.Module, optimizer_kwargs: hf_parse.OptimizerKwargs
) -> Union[hf_opt.Adafactor, hf_opt.AdamW]:
    """
    This follows the function in transformer's Trainer to construct the optimizer.

    Args:
        model: model whose parameters will be updated by the optimizer
        weight_decay: weight_decay factor to apply to weights
        optimizer_kwargs: see OptimizerKwargs in _config_parser.py for expected fields
    Returns:
        optimizer configured accordingly
    """
    optimizer_grouped_parameters = group_parameters_for_optimizer(
        model, optimizer_kwargs.weight_decay
    )
    if optimizer_kwargs.adafactor:
        return hf_opt.Adafactor(
            optimizer_grouped_parameters,
            lr=optimizer_kwargs.learning_rate,
            scale_parameter=optimizer_kwargs.scale_parameter,
            relative_step=optimizer_kwargs.relative_step,
        )
    return hf_opt.AdamW(
        optimizer_grouped_parameters,
        lr=optimizer_kwargs.learning_rate,
        betas=(optimizer_kwargs.adam_beta1, optimizer_kwargs.adam_beta2),
        eps=optimizer_kwargs.adam_epsilon,
    )


def build_default_lr_scheduler(
    optimizer: torch.optim.Optimizer,
    scheduler_kwargs: hf_parse.LRSchedulerKwargs,
) -> Any:
    """
    This follows the function in transformer's Trainer to construct the lr_scheduler.

    Args:
        optimizer: optimizer to apply lr_scheduler to
        scheduler_kwargs: see LRSchedulerKwargs in _config_parser.py for expected fields.
    Returns:
        lr_scheduler configured accordingly
    """
    return hf_opt.get_scheduler(
        scheduler_kwargs.lr_scheduler_type,
        optimizer,
        num_warmup_steps=scheduler_kwargs.num_warmup_steps,
        num_training_steps=scheduler_kwargs.num_training_steps,
    )


def default_load_dataset(
    data_config: Union[Dict, attrdict.AttrDict]
) -> Union[
    hf_datasets.Dataset,
    hf_datasets.IterableDataset,
    hf_datasets.DatasetDict,
    hf_datasets.IterableDatasetDict,
]:
    """
    Creates the dataset using HuggingFace datasets' load_dataset method.
    If a dataset_name is provided, we will use that long with the dataset_config_name.
    Otherwise, we will create the dataset using provided train_file and validation_file.

    Args:
        data_config: arguments for load_dataset.  See DatasetKwargs for expected fields.
    Returns:
        Dataset returned from hf_datasets.load_dataset.
    """
    (data_config,) = hf_parse.parse_dict_to_dataclasses((hf_parse.DatasetKwargs,), data_config)
    # This method is common in nearly all main HF examples.
    if data_config.dataset_name is not None:
        # Downloading and loading a dataset from the hub.
        datasets = hf_datasets.load_dataset(
            data_config.dataset_name, data_config.dataset_config_name
        )
        assert hasattr(datasets, "keys"), "Expected a dictionary of datasets."
        datasets = cast(Union[hf_datasets.DatasetDict, hf_datasets.IterableDatasetDict], datasets)

        if "validation" not in datasets.keys():
            assert (
                "validation_split_percentage" in data_config
            ), "Validation split not provided by this huggingface dataset. Please specify "
            "validation_split_percentage in data_config for use to create validation set"
            datasets["validation"] = hf_datasets.load_dataset(
                data_config.dataset_name,
                data_config.dataset_config_name,
                split=f"train[:{data_config.validation_split_percentage}%]",
            )
            datasets["train"] = hf_datasets.load_dataset(
                data_config.dataset_name,
                data_config.dataset_config_name,
                split=f"train[{data_config.validation_split_percentage}%:]",
            )
    else:
        data_files = {}
        if data_config.train_file is not None:
            data_files["train"] = data_config.train_file
        if data_config.validation_file is not None:
            data_files["validation"] = data_config.validation_file
        extension = data_config.train_file.split(".")[-1]
        if extension == "txt":
            extension = "text"
        datasets = hf_datasets.load_dataset(extension, data_files=data_files)
    return datasets


class BaseTransformerTrial(det_torch.PyTorchTrial):
    """
    This is the base PyTorchTrial for transformers that implements the ``__init__`` and
    ``train_batch`` methods.

    You can subclass ``BaseTransformerTrial`` to customize a trial for your own usage by filing in
    the expected methods for data loading and evaluation.
    """

    def __init__(self, context: det_torch.PyTorchTrialContext) -> None:

        send_analytics("BaseTransformerTrial Created")
        self.context = context
        # A subclass of BaseTransformerTrial may have already set hparams and data_config
        # attributes so we only reset them if they do not exist.
        if not hasattr(self, "hparams"):
            self.hparams = attrdict.AttrDict(context.get_hparams())
        if not hasattr(self, "data_config"):
            self.data_config = attrdict.AttrDict(context.get_data_config())
        if not hasattr(self, "exp_config"):
            self.exp_config = attrdict.AttrDict(context.get_experiment_config())
        # Check to make sure all expected hyperparameters are set.
        self.check_hparams()

        # Parse hparams and data_config.
        (
            self.config_kwargs,
            self.tokenizer_kwargs,
            self.model_kwargs,
        ) = hf_parse.default_parse_config_tokenizer_model_kwargs(self.hparams)
        optimizer_kwargs, scheduler_kwargs = hf_parse.default_parse_optimizer_lr_scheduler_kwargs(
            self.hparams
        )

        self.config, self.tokenizer, self.model = build_using_auto(
            self.config_kwargs,
            self.tokenizer_kwargs,
            self.hparams.model_mode,
            self.model_kwargs,
            use_pretrained_weights=self.hparams.use_pretrained_weights,
        )
        self.model = self.context.wrap_model(self.model)

        self.optimizer = self.context.wrap_optimizer(
            build_default_optimizer(self.model, optimizer_kwargs)
        )

        if self.hparams.use_apex_amp:
            self.model, self.optimizer = self.context.configure_apex_amp(
                models=self.model,
                optimizers=self.optimizer,
            )

        self.lr_scheduler = self.context.wrap_lr_scheduler(
            build_default_lr_scheduler(self.optimizer, scheduler_kwargs),
            det_torch.LRScheduler.StepMode.STEP_EVERY_BATCH,
        )

        self.grad_clip_fn = None

        if optimizer_kwargs.max_grad_norm > 0:  # type: ignore
            self.grad_clip_fn = lambda x: torch.nn.utils.clip_grad_norm_(
                x, optimizer_kwargs.max_grad_norm
            )

    def check_hparams(self) -> None:
        # We require hparams to be an AttrDict.
        if not isinstance(self.hparams, attrdict.AttrDict):
            self.hparams = attrdict.AttrDict(self.hparams)

        if "num_training_steps" not in self.hparams:
            # Compute the total number of training iterations used to configure the
            # learning rate scheduler.
            self.hparams.num_training_steps = model_hub.utils.compute_num_training_steps(
                self.context.get_experiment_config(), self.context.get_global_batch_size()
            )
        if "use_pretrained_weights" not in self.hparams:
            logging.warning(
                "We will be using pretrained weights for the model by default."
                "If you want to train the model from scratch, you can set a hyperparameter "
                "named use_pretrained_weights to false in the experiment config."
            )
            self.hparams.use_pretrained_weights = True

        required_hps = ("use_apex_amp", "model_mode", "num_training_steps")
        for hp in required_hps:
            assert (
                hp in self.hparams
            ), "{} is a required hyperparameter for BaseTransformerTrial".format(hp)

    def train_batch(self, batch: Any, epoch_idx: int, batch_idx: int) -> Any:
        # By default, all HF models return the loss in the first element.
        # We do not automatically apply a label smoother for the user.
        # If this is something you want to use, please see how it's
        # applied by transformers.Trainer:
        # https://github.com/huggingface/transformers/blob/v4.3.3/src/transformers/trainer.py#L1324
        outputs = self.model(**batch)
        loss = outputs["loss"] if isinstance(outputs, dict) else outputs[0]
        self.context.backward(loss)
        self.context.step_optimizer(self.optimizer, self.grad_clip_fn)
        return loss
