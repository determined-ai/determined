import dataclasses
import logging
from typing import Any, Dict, Optional, Tuple, Union

import attrdict

logger = logging.getLogger(__name__)


@dataclasses.dataclass
class DatasetKwargs:
    dataset_name: Optional[str] = dataclasses.field(
        default=None,
        metadata={
            "help": "Path argument to pass to HuggingFace datasets's load_dataset. Can be a "
            "dataset identifier in HuggingFace Datasets Hub or a local path to processing "
            "script."
        },
    )
    dataset_config_name: Optional[str] = dataclasses.field(
        default=None,
        metadata={
            "help": "The name of the dataset configuration to pass to HuggingFace dataset's "
            "load_dataset."
        },
    )
    validation_split_percentage: Optional[float] = dataclasses.field(
        default=None,
        metadata={
            "help": "This is used to create a validation split from the training data when "
            "a dataset does not have a predefined validation split."
        },
    )
    train_file: Optional[str] = dataclasses.field(
        default=None,
        metadata={
            "help": "Path to training data.  This will be used if a dataset_name is not provided."
        },
    )
    validation_file: Optional[str] = dataclasses.field(
        default=None,
        metadata={
            "help": "Path to validation data.  This will be used if a dataset_name is not "
            "provided."
        },
    )


@dataclasses.dataclass
class ConfigKwargs:
    pretrained_model_name_or_path: str = dataclasses.field()
    cache_dir: Optional[str] = dataclasses.field(
        default=None,
        metadata={
            "help": "Where do you want to store the pretrained models downloaded from "
            "huggingface.co"
        },
    )
    revision: Optional[str] = dataclasses.field(
        default="main",
        metadata={
            "help": "The specific model version to use (can be a branch name, tag name or "
            "commit id)."
        },
    )
    use_auth_token: Optional[bool] = dataclasses.field(
        default=False,
        metadata={
            "help": "Will use the token generated when running `transformers-cli login` "
            "(necessary to use this script with private models)."
        },
    )
    num_labels: Optional[int] = dataclasses.field(
        default=2,
        metadata={
            "help": "Number of labels to use in the last layer added to the model, typically "
            "for a classification task."
        },
    )
    finetuning_task: Optional[str] = dataclasses.field(
        default=None,
        metadata={
            "help": "Name of the task used to fine-tune the model. This can be used when "
            "converting from an original PyTorch checkpoint."
        },
    )


@dataclasses.dataclass
class TokenizerKwargs:
    pretrained_model_name_or_path: str = dataclasses.field()
    cache_dir: Optional[str] = dataclasses.field(
        default=None,
        metadata={
            "help": "Where do you want to store the pretrained models downloaded from "
            "huggingface.co"
        },
    )
    use_fast: Optional[bool] = dataclasses.field(
        default=True,
        metadata={
            "help": "Whether to use one of the fast tokenizer (backed by the tokenizers library) "
            "or not."
        },
    )
    revision: Optional[str] = dataclasses.field(
        default="main",
        metadata={
            "help": "The specific model version to use (can be a branch name, tag name or "
            "commit id)."
        },
    )
    use_auth_token: Optional[bool] = dataclasses.field(
        default=False,
        metadata={
            "help": "Will use the token generated when running `transformers-cli login` "
            "(necessary to use this script with private models)."
        },
    )


@dataclasses.dataclass
class ModelKwargs:
    pretrained_model_name_or_path: str = dataclasses.field()
    cache_dir: Optional[str] = dataclasses.field(
        default=None,
        metadata={
            "help": "Where do you want to store the pretrained models downloaded from "
            "huggingface.co"
        },
    )
    revision: Optional[str] = dataclasses.field(
        default="main",
        metadata={
            "help": "The specific model version to use (can be a branch name, tag name or "
            "commit id)."
        },
    )
    use_auth_token: Optional[bool] = dataclasses.field(
        default=False,
        metadata={
            "help": "Will use the token generated when running `transformers-cli login` "
            "(necessary to use this script with private models)."
        },
    )


@dataclasses.dataclass
class OptimizerKwargs:
    weight_decay: Optional[float] = dataclasses.field(
        default=0,
    )
    adafactor: Optional[bool] = dataclasses.field(
        default=False,
        metadata={"help": "Whether to use adafactor optimizer.  Will use AdamW by default."},
    )
    learning_rate: Optional[float] = dataclasses.field(
        default=5e-5,
    )
    max_grad_norm: Optional[float] = dataclasses.field(
        default=1.0,
    )
    adam_beta1: Optional[float] = dataclasses.field(
        default=0.9,
    )
    adam_beta2: Optional[float] = dataclasses.field(
        default=0.999,
    )
    adam_epsilon: Optional[float] = dataclasses.field(
        default=1e-8,
    )
    scale_parameter: Optional[bool] = dataclasses.field(
        default=False,
        metadata={
            "help": "For adafactor optimizer, if True, learning rate is scaled by "
            "root mean square."
        },
    )
    relative_step: Optional[bool] = dataclasses.field(
        default=False,
        metadata={
            "help": "For adafactor optimizer, if True, time-dependent learning rate is computed "
            "instead of external learning rate."
        },
    )


@dataclasses.dataclass
class LRSchedulerKwargs:
    num_training_steps: int = dataclasses.field()
    lr_scheduler_type: Optional[str] = dataclasses.field(
        default="linear",
        metadata={
            "help": "One of linear, cosine, cosine_with_restarts, polynomial, constant, or "
            "constant_with_warmup."
        },
    )
    num_warmup_steps: Optional[int] = dataclasses.field(
        default=0,
    )


def parse_dict_to_dataclasses(
    dataclass_types: Tuple[Any, ...], args: Union[Dict, attrdict.AttrDict], as_dict: bool = False
) -> Tuple[Any, ...]:
    """
    This function will fill in values for a dataclass if the target key is found
    in the provided args dictionary.  We can have one argument key value be filled in
    to multiple dataclasses if the key is found in them.

    Args:
        dataclass_types: dataclasses with expected attributes.
        args: arguments that will be parsed to each of the dataclass_types.
        as_dict: if true will return dictionary instead of AttrDict

    Returns:
        One dictionary for each dataclass with keys filled in from args if found.
    """
    outputs = []
    for dtype in dataclass_types:
        keys = {f.name for f in dataclasses.fields(dtype) if f.init}
        inputs = {k: v for k, v in args.items() if k in keys}
        obj = dtype(**inputs)
        if as_dict:
            obj = attrdict.AttrDict(dataclasses.asdict(obj))
        outputs.append(obj)
    return (*outputs,)


def default_parse_config_tokenizer_model_kwargs(
    hparams: Union[Dict, attrdict.AttrDict]
) -> Tuple[Dict, Dict, Dict]:
    """
    This function will provided hparams into fields for the transformers config, tokenizer,
    and model. See the defined dataclasses ConfigKwargs, TokenizerKwargs, and ModelKwargs for
    expected fields and defaults.

    Args:
        hparams: hyperparameters to parse.

    Returns:
        One dictionary each for the config, tokenizer, and model.
    """
    if not isinstance(hparams, attrdict.AttrDict):
        hparams = attrdict.AttrDict(hparams)
    if "num_labels" not in hparams:
        logger.warning(
            "The num_labels field is not set in hyperparameters. It will be set to 2 by default."
        )
    config_args, tokenizer_args, model_args = parse_dict_to_dataclasses(
        (ConfigKwargs, TokenizerKwargs, ModelKwargs), hparams, as_dict=True
    )

    # If a pretrained_model_name_or_path is provided it will be parsed to the
    # arguments for config, tokenizer, and model.  Then, if specific names are
    # provided for config, tokenizer, or model we will override it.
    if "config_name" in hparams:
        config_args.pretrained_model_name_or_path = hparams.config_name
    if "tokenizer_name" in hparams:
        tokenizer_args.pretrained_model_name_or_path = hparams.tokenizer_name
    if "model_name" in hparams:
        model_args.pretrained_model_name_or_path = hparams.model_name
    assert (
        config_args.pretrained_model_name_or_path is not None
        and tokenizer_args.pretrained_model_name_or_path is not None
        and model_args.pretrained_model_name_or_path is not None
    )
    return config_args, tokenizer_args, model_args


def default_parse_optimizer_lr_scheduler_kwargs(
    hparams: Union[Dict, attrdict.AttrDict]
) -> Tuple[OptimizerKwargs, LRSchedulerKwargs]:
    """
    Parse hparams relevant for the optimizer and lr_scheduler and fills in with
    the same defaults as those used by the transformers Trainer.  See the defined dataclasses
    OptimizerKwargs and LRSchedulerKwargs for expected fields and defaults.

    Args:
        hparams: hparams to parse.

    Returns:
        Configuration for the optimizer and lr scheduler.
    """
    optimizer_args, scheduler_args = parse_dict_to_dataclasses(
        (OptimizerKwargs, LRSchedulerKwargs), hparams
    )
    return optimizer_args, scheduler_args
