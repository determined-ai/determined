import dataclasses
from typing import Any, Dict, Optional, Tuple, Union

import attrdict


class FlexibleDataclass:
    """
    A variant of dataclass that allows fields without defaults to be unpopulated for
    class instances.

    Fields with defaults will always be set as instance attributes.
    Fields without defaults will be set as attributes only if a value is provided in the init.
    """

    def __init__(self, **kwargs: Dict[str, Any]) -> None:
        field_names = [f.name for f in dataclasses.fields(self)]

        # If a dictionary key corresponds to a field, set it as an attribute.
        for k, v in kwargs.items():
            if k in field_names:
                setattr(self, k, v)

        # Otherwise, for fields with defaults, set those attributes.
        for f in dataclasses.fields(self):
            if not hasattr(self, f.name) and f.default is not dataclasses.MISSING:
                setattr(self, f.name, f.default)

    def as_dict(self) -> Dict[str, Any]:
        output = {}
        for f in dataclasses.fields(self):
            if hasattr(self, f.name):
                output[f.name] = getattr(self, f.name)
        return output

    def __repr__(self) -> str:
        fields_str = ", ".join(
            [
                "{}={}".format(f.name, getattr(self, f.name))
                for f in dataclasses.fields(self)
                if hasattr(self, f.name)
            ]
        )
        return self.__class__.__qualname__ + f"({fields_str})"


@dataclasses.dataclass(init=False, repr=False)
class DatasetKwargs(FlexibleDataclass):
    """
    Config parser for dataset fields.

    Either ``dataset_name`` needs to be provided or ``train_file`` and ``validation_file`` need
    to be provided.

    Args:
        dataset_name (optional, defaults to ``None``): Path argument to pass to HuggingFace
            ``datasets.load_dataset``. Can be a dataset identifier in HuggingFace Datasets Hub or
            a local path to processing script.
        dataset_config_name (optional, defaults to ``None``): The name of the dataset configuration
            to pass to HuggingFace ``datasets.load_dataset``.
        validation_split_percentage (optional, defaults to ``None``): This is used to create a
            validation split from the training data when a dataset does not have a predefined
            validation split.
        train_file (optional, defaults to ``None``): Path to training data.  This will be used if
            a dataset_name is not provided.
        validation_file (optional, defaults to ``None``): Path to validation data.  This will be
            used if a dataset_name is not provided.

    Returns:
        dataclass with the above fields populated according to provided config.
    """

    dataset_name: Optional[str] = dataclasses.field(
        default=None,
    )
    dataset_config_name: Optional[str] = dataclasses.field(
        default=None,
    )
    validation_split_percentage: Optional[float] = dataclasses.field(
        default=None,
    )
    train_file: Optional[str] = dataclasses.field(
        default=None,
    )
    validation_file: Optional[str] = dataclasses.field(
        default=None,
    )


@dataclasses.dataclass(init=False, repr=False)
class ConfigKwargs(FlexibleDataclass):
    """
    Config parser for transformers config fields.

    Args:
        pretrained_model_name_or_path: Path to pretrained model or model identifier from
            huggingface.co/models.
        cache_dir (optional, defaults to ``None``): Where do you want to store the pretrained models
            downloaded from huggingface.co.
        revision (optional, defaults to ``None``): The specific model version to use (can be a
            branch name, tag name or commit id).
        use_auth_token (optional, defaults to ``None``): Will use the token generated when running
            ``transformers-cli login`` (necessary to use this script with private models).
        num_labels (optional, excluded if not provided): Number of labels to use in the last layer
            added to the model, typically for a classification task.
        finetuning_task (optional, excluded if not provided): Name of the task used to fine-tune
            the model. This can be used when converting from an original PyTorch checkpoint.

    Returns:
        dataclass with the above fields populated according to provided config.
    """

    # Fields without defaults will be set as attributes only if a value is provided in the init.
    num_labels: Optional[int] = dataclasses.field()
    finetuning_task: Optional[str] = dataclasses.field()

    # Fields with defaults should always be set.
    pretrained_model_name_or_path: Optional[str] = dataclasses.field(
        default=None,
    )
    cache_dir: Optional[str] = dataclasses.field(
        default=None,
    )
    revision: Optional[str] = dataclasses.field(
        default="main",
    )
    use_auth_token: Optional[bool] = dataclasses.field(
        default=False,
    )


@dataclasses.dataclass(init=False, repr=False)
class TokenizerKwargs(FlexibleDataclass):
    """
    Config parser for transformers tokenizer fields.

    Args:
        pretrained_model_name_or_path: Path to pretrained model or model identifier from
            huggingface.co/models.
        cache_dir (optional, defaults to ``None``): Where do you want to store the pretrained models
            downloaded from huggingface.co.
        revision (optional, defaults to ``None``): The specific model version to use (can be a
            branch name, tag name or commit id).
        use_auth_token (optional, defaults to ``None``): Will use the token generated when running
            ``transformers-cli login`` (necessary to use this script with private models).
        use_fast (optional, defaults to ``True``): Whether to use one of the fast tokenizer
            (backed by the tokenizers library) or not.
        do_lower_case (optional, excluded if not provided): Indicate if tokenizer should do lower
            case

    Returns:
        dataclass with the above fields populated according to provided config.

    """

    # Fields without defaults will be set as attributes only if a value is provided in the init.
    do_lower_case: Optional[bool] = dataclasses.field()

    # Fields with defaults should always be set.
    pretrained_model_name_or_path: Optional[str] = dataclasses.field(
        default=None,
    )
    cache_dir: Optional[str] = dataclasses.field(
        default=None,
    )
    revision: Optional[str] = dataclasses.field(
        default="main",
    )
    use_auth_token: Optional[bool] = dataclasses.field(
        default=False,
    )
    use_fast: Optional[bool] = dataclasses.field(
        default=True,
    )


@dataclasses.dataclass
class ModelKwargs(FlexibleDataclass):
    """
    Config parser for transformers model fields.

    Args:
        pretrained_model_name_or_path: Path to pretrained model or model identifier from
            huggingface.co/models.
        cache_dir (optional, defaults to ``None``): Where do you want to store the pretrained models
            downloaded from huggingface.co.
        revision (optional, defaults to ``None``): The specific model version to use (can be a
            branch name, tag name or commit id).
        use_auth_token (optional, defaults to ``None``): Will use the token generated when running
            ``transformers-cli login`` (necessary to use this script with private models).

    Returns:
        dataclass with the above fields populated according to provided config.

    """

    pretrained_model_name_or_path: str = dataclasses.field()
    cache_dir: Optional[str] = dataclasses.field(
        default=None,
    )
    revision: Optional[str] = dataclasses.field(
        default="main",
    )
    use_auth_token: Optional[bool] = dataclasses.field(
        default=False,
    )


@dataclasses.dataclass
class OptimizerKwargs:
    """
    Config parser for transformers optimizer fields.

    """

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
    """
    Config parser for transformers lr scheduler fields.
    """

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
    dataclass_types: Tuple[Any, ...],
    args: Union[Dict[str, Any], attrdict.AttrDict],
    as_dict: bool = False,
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
            try:
                obj = attrdict.AttrDict(obj.as_dict())
            except AttributeError:
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
