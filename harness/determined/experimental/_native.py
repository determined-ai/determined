import logging
import pathlib
import sys
import tempfile
import warnings
from typing import Any, Dict, List, Optional, Tuple, Type

import determined as det
import determined.common
from determined import constants, errors, load, workload
from determined.common import api, check, context, util
from determined.common.api import authentication


def _in_ipython() -> bool:
    import __main__

    if hasattr(__main__, "__file__"):
        return False
    try:
        import IPython

        IPython
    except ImportError:
        return False
    return True


def _get_current_args() -> List:
    return sys.argv[1:]


def _set_command_default(
    context_dir: pathlib.Path, command: Optional[List[str]] = None
) -> List[str]:
    if not command or len(command) == 0:
        if _in_ipython():
            raise errors.InvalidExperimentException(
                "Must specify the location of the notebook file "
                "relative to the context directory when in notebook."
            )

        exp_path = pathlib.Path(sys.argv[0]).resolve()
        exp_rel_path = exp_path.relative_to(context_dir.resolve())
        if exp_rel_path.suffix in {"py", "ipynb"}:
            raise errors.InvalidExperimentException(
                "Command must begin with a file with the suffix .py or .ipynb. "
                "Found {}".format(command)
            )

        command = [str(exp_rel_path), *_get_current_args()]

    return command


def _submit_experiment(
    config: Optional[Dict[str, Any]],
    context_dir: str,
    command: Optional[List[str]],
    test: bool = False,
    master_url: Optional[str] = None,
) -> int:
    if context_dir == "":
        raise errors.InvalidExperimentException("Cannot specify the context directory to be empty.")

    context_path = pathlib.Path(context_dir)
    config = {**constants.DEFAULT_EXP_CFG, **(config or {})}
    config.setdefault("internal", {})
    config["internal"]["native"] = {"command": _set_command_default(context_path, command)}
    logging.info(f"Creating an experiment with config: {config}")

    if master_url is None:
        master_url = util.get_default_master_address()

    exp_context = context.Context.from_local(context_path)

    # When a requested_user isn't specified to initialize_session(), the
    # authentication module will attempt to use the token store to grab the
    # current logged-in user. If there is no logged in user found, it will
    # default to constants.DEFAULT_DETERMINED_USER.
    # TODO: refactor this to use the python sdk rather than using the cli_auth singleton.
    authentication.cli_auth = authentication.Authentication(
        master_url, requested_user=None, try_reauth=True
    )

    if test:
        return api.create_test_experiment_and_follow_logs(master_url, config, exp_context)
    else:
        return api.create_experiment_and_follow_logs(master_url, config, exp_context)


def _make_test_workloads(
    checkpoint_dir: pathlib.Path, config: det.ExperimentConfig
) -> workload.Stream:
    interceptor = workload.WorkloadResponseInterceptor()

    logging.info("Training one batch")
    yield from interceptor.send(workload.train_workload(1), [])
    metrics = interceptor.metrics_result()
    batch_metrics = metrics["metrics"]["batch_metrics"]
    check.eq(len(batch_metrics), config.scheduling_unit())
    logging.info(f"Finished training, metrics: {batch_metrics}")

    logging.info("Validating one batch")
    yield from interceptor.send(workload.validation_workload(1), [])
    validation = interceptor.metrics_result()
    v_metrics = validation["metrics"]["validation_metrics"]
    logging.info(f"Finished validating, validation metrics: {v_metrics}")

    logging.info(f"Saving a checkpoint to {checkpoint_dir}.")
    yield workload.checkpoint_workload(), [checkpoint_dir], workload.ignore_workload_response
    logging.info(f"Finished saving a checkpoint to {checkpoint_dir}.")

    yield workload.terminate_workload(), [], workload.ignore_workload_response
    logging.info("The test experiment passed.")


def _load_trial_for_checkpoint_export(
    context_dir: pathlib.Path,
    managed_training: bool,
    config: Dict[str, Any],
    hparams: Dict[str, Any],
) -> Tuple[Type[det.Trial], det.TrialContext]:
    with det._local_execution_manager(context_dir):
        trial_class = load.trial_class_from_entrypoint(config["entrypoint"])
        env, rendezvous_info, hvd_config = det._make_local_execution_env(
            managed_training=managed_training, test_mode=False, config=config, hparams=hparams
        )
        trial_context = trial_class.trial_context_class(env, hvd_config, rendezvous_info)
    return trial_class, trial_context


def test_one_batch(
    trial_class: Type[det.Trial],
    config: Optional[Dict[str, Any]] = None,
) -> Any:
    # Override the scheduling_unit value to 1.
    config = {**(config or {}), "scheduling_unit": 1}

    logging.info("Running a minimal test experiment locally")
    checkpoint_dir = tempfile.TemporaryDirectory()
    env, rendezvous_info, hvd_config = det._make_local_execution_env(
        managed_training=True, test_mode=True, config=config, limit_gpus=1
    )
    workloads = _make_test_workloads(
        pathlib.Path(checkpoint_dir.name).joinpath("checkpoint"), env.experiment_config
    )
    logging.info(f"Using hyperparameters: {env.hparams}.")
    logging.debug(f"Using a test experiment config: {env.experiment_config}.")

    # Case 2: test one batch for Trial implementation.
    controller = load.load_trial(
        trial_class=trial_class,
        env=env,
        workloads=workloads,
        load_path=None,
        rendezvous_info=rendezvous_info,
        hvd_config=hvd_config,
    )
    controller.run()
    checkpoint_dir.cleanup()
    logging.info("Note: to submit an experiment to the cluster, change local parameter to False")


def create(
    trial_def: Type[det.Trial],
    config: Optional[Dict[str, Any]] = None,
    local: bool = False,
    test: bool = False,
    context_dir: str = "",
    command: Optional[List[str]] = None,
    master_url: Optional[str] = None,
) -> Any:
    # TODO: Add a reference to the local development tutorial.
    """
    Create an experiment.

    Arguments:
        trial_def:
            A class definition implementing the :class:`determined.Trial`
            interface.

        config:
            A dictionary representing the experiment configuration to be
            associated with the experiment.

        local:
            A boolean indicating if training should be done locally. When
            ``False``, the experiment will be submitted to the Determined
            cluster. Defaults to ``False``.

        test:
            A boolean indicating if the experiment should be shortened
            to a minimal loop of training on a small amount of data,
            performing validation, and checkpointing.  ``test=True`` is
            useful for quick iteration during model porting or debugging
            because common errors will surface more quickly.  Defaults
            to ``False``.

        context_dir:
            A string filepath that defines the context directory. All model
            code will be executed with this as the current working directory.

            When ``local=False``, this argument is required. All files in this
            directory will be uploaded to the Determined cluster. The total
            size of this directory must be under 96 MB.

            When ``local=True``, this argument is optional and defaults to
            the current working directory.

        command:
            A list of strings that is used as the entrypoint of the training
            script in the Determined task environment. When executing this
            function via a Python script, this argument is inferred to be
            ``sys.argv`` by default. When executing this function via IPython
            or Jupyter notebook, this argument is required.

            Example: When creating an experiment by running ``python train.py
            --flag value``, the default command is inferred as ``["train.py",
            "--flag", "value"]``.

        master_url:
            An optional string to use as the Determined master URL when
            ``local=False``. If not specified, will be inferred from the
            environment variable ``DET_MASTER``.
    """

    if local and not test:
        raise NotImplementedError(
            "det.create(local=True, test=False) is not yet implemented. Please set local=False "
            "or test=True."
        )

    determined.common.set_logger(
        util.debug_mode() or det.ExperimentConfig(config or {}).debug_enabled()
    )

    if local:
        # Local test mode.
        with det._local_execution_manager(pathlib.Path(context_dir).resolve()):
            return test_one_batch(
                trial_class=trial_def,
                config=config,
            )

    elif not load.RunpyGlobals.is_initialized():
        # Cluster mode, but still running locally; submit the experiment.
        _submit_experiment(
            config=config,
            test=test,
            context_dir=context_dir,
            command=command,
            master_url=master_url,
        )

    else:
        # Cluster mode, now on the cluster; actually train.
        load.RunpyGlobals.set_runpy_trial_result(trial_def)
        raise det.errors.StopLoadingImplementation()


def create_trial_instance(
    trial_def: Type[det.Trial],
    checkpoint_dir: str,
    config: Optional[Dict[str, Any]] = None,
    hparams: Optional[Dict[str, Any]] = None,
) -> det.Trial:
    """
    Deprecated: please use your TrialContext's .from_config() method instead.

    Create a trial instance from a Trial class definition. This can be a useful
    utility for debugging your trial logic in any development environment.

    Arguments:
        trial_def: A class definition that inherits from the det.Trial interface.
        checkpoint_dir:
            The checkpoint directory that the trial will use for loading and
            saving checkpoints.
        config:
            An optional experiment configuration that is used to initialize the
            :class:`determined.TrialContext`. If not specified, a minimal default
            is used.
    """
    warnings.warn(
        "det.experimental.create_trial_instance() is now deprecated.  Please use\n"
        "your TrialContext's .from_config() method instead.  Example\n"
        "\n"
        "    context = PyTorchTrialContext.from_config()\n"
        "    my_trial = MyPyTorchTrial(context)\n",
        FutureWarning,
    )
    determined.common.set_logger(
        util.debug_mode() or det.ExperimentConfig(config or {}).debug_enabled()
    )
    env, rendezvous_info, hvd_config = det._make_local_execution_env(
        managed_training=False, test_mode=False, config=config, hparams=hparams
    )
    trial_context = trial_def.trial_context_class(env, hvd_config, rendezvous_info=rendezvous_info)
    return trial_def(trial_context)
