import json
import logging
import pathlib
import warnings
from typing import Any, Dict, Optional, Tuple, Type, cast

import torch

import determined as det
from determined import core, errors, load, pytorch, util

logger = logging.getLogger("determined.pytorch")


class CheckpointLoadContext(pytorch.PyTorchTrialContext):
    """
    CheckpointLoadContext is a special PyTorchTrialContext that can be used to load Trial classes
    outside of normal training loops.

    It does not support actually reporting metrics to a real master or uploading checkpoints or any
    of the normal behaviors associated with model training.  :func:`determined.pytorch.init()`
    should still be used for normal training.

    CheckpointLoadContext is meant to be used by users using the PyTorchTrial Trainer directly.
    Users using the Trainer might prefer CheckpointLoadContext because it allows them to create a
    Trial class with extra parameters they may have added.

    Users who are relying on the legacy `entrypoint: my_model:MyTrainer` way of launching their code
    should continue to use :func:`~determined.pytorch.load_trial_from_checkpoint_path()`.
    Example usage:

    .. code:: python

       import determined as det
       from determined import pytorch
       from determined.experimental import client

       # Download checkpoint and load training code from checkpoint.
       path = client.get_checkpoint(MY_UUID)
       with det.import_from_path(path + "/code/"):
           import my_model_def

       # Create CheckpointLoadContext for instantiating trial.
       context = pytorch.CheckpointLoadContext()

       # Instantiate trial with context and any other args.
       my_trial = my_model_def.MyTrial(context, ...)
    """

    def __init__(
        self,
        hparams: Optional[Dict] = None,
        exp_conf: Optional[Dict[str, Any]] = None,
    ) -> None:
        _, container_gpus, _ = det._get_gpus(limit_gpus=False)
        super().__init__(
            core_context=core._dummy_init(),
            trial_seed=0,
            hparams=hparams,
            slots_per_trial=1,
            num_gpus=len(container_gpus),
            exp_conf=exp_conf,
            aggregation_frequency=1,
            steps_completed=0,
            managed_training=False,
            debug_enabled=False,
        )


def load_trial_from_checkpoint_path(
    path: str,
    trial_class: Optional[Type[pytorch.PyTorchTrial]] = None,
    trial_kwargs: Optional[Dict[str, Any]] = None,
    torch_load_kwargs: Optional[Dict[str, Any]] = None,
    **kwargs: Dict[str, Any],
) -> pytorch.PyTorchTrial:
    """
    Loads a checkpoint written by a PyTorchTrial.

    You should have already downloaded the checkpoint files, likely with
    :meth:`Checkpoint.download() <determined.experimental.client.Checkpoint.download()>`.

    The return value will be a restored instance of the subclass PyTorchTrial you used for training.

    Arguments:
        path (string): Top level directory to load the checkpoint from.
        trial_class (optional): Provide your PyTorchTrial class to be loaded.  Only necessary if
            the automatic import logic is insufficient.
        trial_kwargs (optional): Additional keyword arguments to be passed to your PyTorchTrial
            class, in addition to the context, which will always be the first positional parameter.
        torch_load_kwargs (optional): Keyword arguments for ``torch.load``. See documentation for
            `torch.load
            <https://pytorch.org/docs/stable/torch.html?highlight=torch%20load#torch.load>`_.
        **kwargs (deprecated): Use torch_load_kwargs instead.
    """
    trial_kwargs = trial_kwargs or {}
    torch_load_kwargs = torch_load_kwargs or {}
    if kwargs:
        if torch_load_kwargs:
            raise ValueError("kwargs may not be set if torch_load_kwargs is also set")
        warnings.warn(
            "Passing **kwargs to load_trial_from_checkpoint_path() is deprecated, and support for "
            "doing so will be removed in the future.  Please pass a dict of kwargs as "
            "`torch_load_kwargs` instead.",
            FutureWarning,
            stacklevel=2,
        )
        torch_load_kwargs = kwargs

    ckpt_dir = pathlib.Path(path)
    load_data_path = ckpt_dir.joinpath("load_data.json")
    metadata_path = ckpt_dir.joinpath("metadata.json")
    # If the user used the Trainer API directly (not via the exec/harness.py legacy shims) then we
    # will take care to avoid the same shims during checkpoint loading.
    is_trainer = False
    # With Trainer API, if the Trial was defined in __main__, we had to guess how to import it
    # automatically, and we log extra messages so the user knows why it might fail.
    trial_in_main = False
    if load_data_path.exists():
        # PyTorchTrial.build_model() was an old api that was disallowed after 0.14.0, and
        # load_data.json indicates the checkpoint is much newer than that.
        detect_build_model = False
        with load_data_path.open() as f:
            load_data = json.load(f)
        if load_data["trial_type"] != "PyTorchTrial":
            logger.warning(
                "Checkpoint does not appear to be a valid PyTorchTrial checkpoint, "
                "continuing anyway..."
            )
        experiment_config = load_data["experiment_config"]
        hparams = load_data["hparams"]
        trial_cls_spec = load_data["trial_cls_spec"]
        is_trainer = load_data.get("is_trainer", False)
        trial_in_main = load_data.get("trial_in_main", False)

    elif metadata_path.exists():
        # PyTorchTrial.build_model() was an old api that was disallowed after 0.14.0.  There's a
        # small chance that this model is that old.
        detect_build_model = True
        # Old checkpoints (<=0.17.7) used to depend on metadata coming from the master in
        # Checkpoint.download().
        with metadata_path.open() as f:
            metadata = json.load(f)
        is_torch = metadata.get("framework", "").startswith("torch")
        has_torch_version = "torch_version" in metadata
        # Older metadata layout contained torch_version and tensorflow_version
        # as keys. Eventually, we should drop support for the older format.
        if not is_torch and not has_torch_version:
            logger.warning(
                "Checkpoint does not appear to be a valid PyTorchTrial checkpoint, "
                "continuing anyway..."
            )
        experiment_config = metadata["experiment_config"]
        hparams = metadata["hparams"]
        # When this format was in use, all entrypoints were trial classes.
        trial_cls_spec = experiment_config["entrypoint"]
    else:
        raise AssertionError(
            "Checkpoint does not have either load_data.json or metadata.json.  Checkpoints written "
            "by Determined 0.17.7 and earlier did not save enough information to be loaded "
            "directly from the files in the checkpoint.  Instead, a metadata.json was written "
            "during the call to Checkpoint.download().  If you are reading an old checkpoint "
            "directly from checkpoint storage, you can either use Checkpoint.download() instead or "
            "you can use Checkpoint.write_metadata_file('metadata.json') to create a suitable "
            "metadata file for loading a legacy checkpoint."
        )

    if trial_in_main:
        module, _, qualname = trial_cls_spec.partition(":")
        if module == "":
            raise ValueError(
                f"Unable to load trial class {qualname}, which was defined in the entrypoint "
                "script, but the entrypoint script did not appear to be an importable module. "
                "Define your Trial class in an importable module instead."
            )
        logger.warning(
            f"Importing trial class {qualname}, which was defined in the entrypoint script. "
            f"Assuming that entrypoint is importable via `import {module}`.  Otherwise, define "
            "your Trial class in an importable file."
        )

    if not is_trainer:
        # Indirect usage of the Trainer API should always have experiment_config and hparams set.
        assert experiment_config is not None and hparams is not None
        trial_class, trial_context = _load_pytorch_trial_with_shims(
            ckpt_dir.joinpath("code"),
            managed_training=False,
            trial_cls_spec=trial_cls_spec,
            config=experiment_config,
            hparams=hparams,
        )
    else:
        # Users using the Trainer API directly should not have any legacy shims.
        if trial_class is None:
            try:
                with det.import_from_path(ckpt_dir.joinpath("code")):
                    trial_class = load.trial_class_from_entrypoint(trial_cls_spec)  # type: ignore
            except Exception as e:
                raise ValueError(
                    f"Automatic import logic failed to import Trial class as {trial_cls_spec}. "
                    "You will need to provide your Trial class via the trial_class argument intead."
                ) from e
        # Load checkpoint without reading anything from the experiment config.
        trial_context = CheckpointLoadContext(hparams, experiment_config)

    trial = trial_class(trial_context, **trial_kwargs)  # type: ignore

    checkpoint = torch.load(  # type: ignore
        str(ckpt_dir.joinpath("state_dict.pth")), **torch_load_kwargs
    )

    # We are still backwards compatible with checkpoints saved in the pre-0.12.13 PyTorchTrial API,
    # but when we can guarantee that the pre-0.12.13 API was not in use, we avoid checking for a
    # .build_model() method, so that users who choose to use that name for their own purposes are
    # unaffected.
    if detect_build_model:
        model_func = util.get_member_func(trial, "build_model")
        if "model_state_dict" in checkpoint:
            # Backward compatible with older checkpoint.
            if model_func is not None:
                model = cast(torch.nn.Module, model_func())
                model.load_state_dict(checkpoint["model_state_dict"])
                # Note, with the very old checkpoints, we actually return the bare model.
                return model  # type: ignore
            raise errors.InvalidCheckpointException()

        # Backward compatible with older checkpoint.
        if model_func is not None:
            model = cast(torch.nn.Module, model_func())
            model.load_state_dict(checkpoint["models_state_dict"][0])
            # Note, with the very old checkpoints, we actually return the bare model.
            return model  # type: ignore

    # Latest model format.
    for idx, model in enumerate(trial_context.models):
        model.load_state_dict(checkpoint["models_state_dict"][idx])
    return trial


def _load_pytorch_trial_with_shims(
    context_dir: pathlib.Path,
    managed_training: bool,
    trial_cls_spec: str,
    config: Dict[str, Any],
    hparams: Dict[str, Any],
) -> Tuple[Type[pytorch.PyTorchTrial], pytorch.PyTorchTrialContext]:
    with det._local_execution_manager(context_dir):
        trial_class = cast(
            Type[pytorch.PyTorchTrial], load.trial_class_from_entrypoint(trial_cls_spec)
        )

        config = det.ExperimentConfig(
            det._make_local_execution_exp_config(
                config, "/tmp", managed_training=managed_training, test_mode=False
            )
        )
        use_gpu, container_gpus, slot_ids = det._get_gpus(limit_gpus=False)
        fp16_compression = bool(
            config.get_optimizations_config().get("gradient_compression", False)
        )
        average_aggregated_gradients = bool(config.average_training_metrics_enabled())

        trial_context = pytorch.PyTorchTrialContext(
            core_context=core._dummy_init(),
            trial_seed=config.experiment_seed(),
            hparams=hparams,
            slots_per_trial=config.slots_per_trial(),
            num_gpus=len(container_gpus),
            exp_conf=config,
            aggregation_frequency=int(
                config.get_optimizations_config().get("aggregation_frequency", 1)
            ),
            steps_completed=0,
            managed_training=managed_training,
            debug_enabled=False,
        )
        trial_context._set_default_gradient_compression(fp16_compression)
        trial_context._set_default_average_aggregated_gradients(average_aggregated_gradients)

    return trial_class, trial_context
