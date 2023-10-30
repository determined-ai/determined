"""Singleton-style Core API."""

import atexit
import dataclasses
import logging
import uuid
from typing import Any, Dict, List, Optional, Union

import determined
from determined import core, experimental
from determined.common import util
from determined.experimental import core_v2

logger = logging.getLogger("determined.experimental.core_v2")


_context = None  # type: Optional[core.Context]
_client = None  # type: Optional[experimental.Determined]
_atexit_registered = False  # type: bool


@dataclasses.dataclass
class DefaultConfig:
    """
    In the future, `DefaultConfig` options will be merged with the experiment config values when
    running in the managed mode.
    """

    name: Optional[str] = None
    hparams: Optional[Dict[str, Any]] = None
    data: Optional[Dict] = None
    description: Optional[str] = None
    labels: Optional[List[str]] = None
    # Also to be added:
    # - hyperparameters: const only
    # - checkpoint_policy: for optional gc
    # For managed mode, workspace and project MUST be present in the exp conf for RBAC reasons.
    # We expose them as separate arguments on the `det.init` call.
    # - workspace
    # - project
    # Unsupported:
    # - bind_mounts
    # - data_layer
    # - debug
    # - entrypoint
    # - environment
    # - internal
    # - max_restarts
    # - optimizations
    # - profiling
    # - reproducibility
    # - security
    # - slurm
    # Searcher:
    # - searcher
    # - min_checkpoint_period
    # - min_validation_period
    # - pbs
    # - perform_initial_validation
    # - records_per_epoch
    # - scheduling_unit
    # Deprecated:
    # - data_layer
    # - pbs
    # - tensorboard_storage
    checkpoint_storage: Optional[Union[str, Dict[str, Any]]] = None
    # Searcher is currently a hack to disambiguate single trial experiments
    # and hp searches in WebUI.
    # Also searcher metric is useful for sorting in the UI and, in the future, checkpoint gc.
    searcher: Optional[Dict[str, Any]] = None
    # TODO(ilia): later to be replaced with:
    # Unmanaged mode only config options:
    # multi_trial_experiment: bool = False
    # metric: Optional[str] = None
    # smaller_is_better: bool = True  # mode?


@dataclasses.dataclass
class UnmanagedConfig:
    """
    `UnmanagedConfig` values are only used in the unmanaged mode.
    """

    # For the managed mode, workspace is critical for RBAC so it cannot be easily
    # merged and patched during the experiment runtime.
    workspace: Optional[str] = None
    project: Optional[str] = None
    # External experiment & trial ids.
    # `external_experiment_id` is used to uniquely identify an experiment when grouping
    # multiple trials as one HP search, or if any trial within this experiment will be resumed.
    # `external_trial_id` is used to uniquely identify the resumed trial within the experiment.
    # if `external_trial_id` is specified, `external_experiment_id` MUST be passed as well.
    #
    # If you are going to resume trials, whether in hp search or single-trial experiments,
    # specify both `external_experiment_id` and `external_trial_id`.
    # If you are going to use hp search, but will not resume trials,
    # specifing the `external_experiment_id` is sufficient.
    # If you are not going to use either feature, omit these options.
    external_experiment_id: Optional[str] = None
    external_trial_id: Optional[str] = None


def _set_globals() -> None:
    """
    global train
    global checkpoint
    global distributed
    global preempt
    global searcher
    global info
    """

    assert _context is not None
    core_v2.train = _context.train
    core_v2.checkpoint = _context.checkpoint
    core_v2.distributed = _context.distributed
    core_v2.preempt = _context.preempt
    core_v2.searcher = _context.searcher
    core_v2.info = _context.info


def _init_context(
    client: experimental.Determined,
    defaults: Optional[DefaultConfig] = None,
    unmanaged: Optional[UnmanagedConfig] = None,
    distributed: Optional[core.DistributedContext] = None,
    checkpoint_storage: Optional[Union[str, Dict[str, Any]]] = None,
    preempt_mode: core.PreemptMode = core.PreemptMode.WorkersAskChief,
    tensorboard_mode: core.TensorboardMode = core.TensorboardMode.AUTO,
) -> core.Context:
    info = determined.get_cluster_info()
    if info is not None and info.task_type == "TRIAL":
        # Managed trials.
        _context = core_v2._make_v2_context(
            distributed=distributed,
            checkpoint_storage=checkpoint_storage,
            preempt_mode=preempt_mode,
            tensorboard_mode=tensorboard_mode,
            client=client,
        )
        return _context

    # Unmanaged trials.
    if defaults is None:
        raise NotImplementedError(
            "either specify `defaults`, or run as a managed determined experiment"
        )

    # Construct the config.
    defaulted_config = defaults or DefaultConfig()
    unmanaged_config = unmanaged or UnmanagedConfig()
    checkpoint_storage = checkpoint_storage or defaulted_config.checkpoint_storage

    config = {
        "name": defaulted_config.name or f"unmanaged-{uuid.uuid4().hex[:8]}",
        "data": defaulted_config.data,
        "description": defaulted_config.description,
        "labels": defaulted_config.labels,
        "searcher": defaulted_config.searcher
        or {
            "name": "single",
            "metric": "unmanaged",
            "max_length": 100000000,
        },
        "workspace": unmanaged_config.workspace,
        "project": unmanaged_config.project,
    }

    config_text = util.yaml_safe_dump(config)
    assert config_text is not None

    unmanaged_info = core_v2._get_or_create_experiment_and_trial(
        client,
        config_text=config_text,
        experiment_id=unmanaged_config.external_experiment_id,
        trial_id=unmanaged_config.external_trial_id,
        distributed=distributed,
        hparams=defaulted_config.hparams,
    )

    _context = core_v2._make_v2_context(
        distributed=distributed,
        checkpoint_storage=checkpoint_storage,
        preempt_mode=preempt_mode,
        tensorboard_mode=tensorboard_mode,
        unmanaged_info=unmanaged_info,
        client=client,
    )
    return _context


def init_context(
    *,
    defaults: Optional[DefaultConfig] = None,
    unmanaged: Optional[UnmanagedConfig] = None,
    client: Optional[experimental.Determined] = None,
    distributed: Optional[core.DistributedContext] = None,
    checkpoint_storage: Optional[Union[str, Dict[str, Any]]] = None,
    preempt_mode: core.PreemptMode = core.PreemptMode.WorkersAskChief,
    tensorboard_mode: core.TensorboardMode = core.TensorboardMode.AUTO,
) -> core.Context:
    """
    Core V2 initializer in the context-manager style.
    """
    if client is None:
        client = experimental.Determined()

    _context = _init_context(
        defaults=defaults,
        unmanaged=unmanaged,
        client=client,
        distributed=distributed,
        checkpoint_storage=checkpoint_storage,
        preempt_mode=preempt_mode,
        tensorboard_mode=tensorboard_mode,
    )

    return _context


def init(
    *,
    defaults: Optional[DefaultConfig] = None,
    unmanaged: Optional[UnmanagedConfig] = None,
    client: Optional[experimental.Determined] = None,
    # Classic core context arguments.
    distributed: Optional[core.DistributedContext] = None,
    checkpoint_storage: Optional[Union[str, Dict[str, Any]]] = None,
    preempt_mode: core.PreemptMode = core.PreemptMode.WorkersAskChief,
    tensorboard_mode: core.TensorboardMode = core.TensorboardMode.AUTO,
    # resume: bool = True  # TODO(ilia): optionally control resume behaviour.
) -> None:
    """
    Core V2 initializer in the singleton style.
    """
    global _context
    global _client
    global _atexit_registered

    if _context is not None:
        _context.close()

    if client is None:
        client = experimental.Determined()
    _client = client

    _context = _init_context(
        defaults=defaults,
        unmanaged=unmanaged,
        client=client,
        distributed=distributed,
        checkpoint_storage=checkpoint_storage,
        preempt_mode=preempt_mode,
        tensorboard_mode=tensorboard_mode,
    )
    _context.start()
    _set_globals()

    if not _atexit_registered:
        atexit.register(close)
        _atexit_registered = True


def close() -> None:
    global _context
    global train

    if _context is not None:
        _context.close()

    _context = None
    core_v2.train = None


def url_reverse_webui_exp_view() -> str:
    assert core_v2.info is not None
    assert core_v2.info._trial_info is not None
    exp_id = core_v2.info._trial_info.experiment_id

    assert _client is not None
    return core_v2._url_reverse_webui_exp_view(_client, exp_id)
