import contextlib
import logging
import math
import os
import pathlib
import random
import sys
import tempfile
from typing import Any, Dict, Iterator, List, Optional, Tuple, Type, cast

import determined as det
import determined_common
import determined_common.api.authentication as auth
from determined import constants, errors, gpu, horovod, load, workload
from determined_common import api, check, context, util


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
) -> Optional[int]:
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
    auth.initialize_session(master_url, requested_user=None, try_reauth=True)

    if test:
        exp_id = api.create_test_experiment(master_url, config, exp_context)
    else:
        exp_id = api.create_experiment(master_url, config, exp_context)
    logging.info(f"Created experiment {exp_id}")

    return exp_id


def _get_gpus() -> Tuple[bool, List[str], List[int]]:
    gpu_ids, gpu_uuids = gpu.get_gpu_ids_and_uuids()
    use_gpu = len(gpu_uuids) > 0
    return use_gpu, gpu_uuids, gpu_ids


def _generate_test_hparam_values(config: Dict[str, Any]) -> Dict[str, Any]:
    def generate_random_value(hparam: Any) -> Any:
        if isinstance(hparam, Dict):
            if hparam["type"] == "const":
                return hparam["val"]
            elif hparam["type"] == "int":
                return random.randint(hparam["minval"], hparam["maxval"])
            elif hparam["type"] == "double":
                return random.uniform(hparam["minval"], hparam["maxval"])
            elif hparam["type"] == "categorical":
                return hparam["vals"][random.randint(0, len(hparam["vals"]) - 1)]
            elif hparam["type"] == "log":
                return math.pow(hparam["base"], random.uniform(hparam["minval"], hparam["maxval"]))
            else:
                raise Exception(f"Wrong type of hyperparameter: {hparam['type']}")
        elif isinstance(hparam, (int, float, str)):
            return hparam
        else:
            raise Exception(f"Wrong type of hyperparameter: {type(hparam)}")

    hparams_def = config.get("hyperparameters", {})
    hparams = {name: generate_random_value(hparams_def[name]) for name in hparams_def}
    return hparams


def _make_test_workloads(
    checkpoint_dir: pathlib.Path, config: det.ExperimentConfig
) -> workload.Stream:
    interceptor = workload.WorkloadResponseInterceptor()

    logging.info("Training one batch")
    yield from interceptor.send(workload.train_workload(1), [1])
    metrics = interceptor.metrics_result()
    batch_metrics = metrics["metrics"]["batch_metrics"]
    check.eq(len(batch_metrics), config.batches_per_step())
    logging.debug(f"Finished training, metrics: {batch_metrics}")

    logging.info("Validating one step")
    yield from interceptor.send(workload.validation_workload(1), [])
    validation = interceptor.metrics_result()
    v_metrics = validation["metrics"]["validation_metrics"]
    logging.debug(f"Finished validating, validation metrics: {v_metrics}")

    logging.info(f"Saving a checkpoint to {checkpoint_dir}.")
    yield workload.checkpoint_workload(), [checkpoint_dir], workload.ignore_workload_response
    logging.info(f"Finished saving a checkpoint to {checkpoint_dir}.")

    yield workload.terminate_workload(), [], workload.ignore_workload_response
    logging.info("The test experiment passed.")


def _make_local_test_experiment_config(input_config: Optional[Dict[str, Any]]) -> Dict[str, Any]:
    """
    Create a local experiment configuration based on an input configuration and
    defaults. Use a shallow merging policy to overwrite our default
    configuration with each entire subconfig specified by a user.

    The defaults and merging logic is not guaranteed to match the logic used by
    the Determined master. This function also does not do experiment
    configuration validation, which the Determined master does.
    """

    input_config = input_config or {}
    config_keys_to_ignore = {
        "bind_mounts",
        "checkpoint_storage",
        "environment",
        "resources",
        "optimizations",
    }
    for key in config_keys_to_ignore:
        if key in input_config:
            logging.info(
                f"'{key}' configuration key is not supported by local test mode and will be ignored"
            )
            del input_config[key]

    return {**constants.DEFAULT_EXP_CFG, **input_config}


def _make_local_test_experiment_env(
    checkpoint_dir: pathlib.Path,
    config: Optional[Dict[str, Any]],
    hparams: Optional[Dict[str, Any]] = None,
) -> Tuple[det.EnvContext, workload.Stream, det.RendezvousInfo, horovod.HorovodContext]:
    config = det.ExperimentConfig(_make_local_test_experiment_config(config))
    hparams = hparams or _generate_test_hparam_values(config)
    use_gpu, container_gpus, slot_ids = _get_gpus()
    local_rendezvous_ports = (
        f"{constants.LOCAL_RENDEZVOUS_PORT},{constants.LOCAL_RENDEZVOUS_PORT+1}"
    )

    env = det.EnvContext(
        master_addr="",
        master_port=1,
        container_id="test_mode",
        experiment_config=config,
        hparams=hparams,
        initial_workload=workload.train_workload(1, 1, 1),
        latest_checkpoint=None,
        use_gpu=use_gpu,
        container_gpus=container_gpus,
        slot_ids=slot_ids,
        debug=config.debug_enabled(),
        workload_manager_type="",
        det_rendezvous_ports=local_rendezvous_ports,
        det_trial_runner_network_interface=constants.AUTO_DETECT_TRIAL_RUNNER_NETWORK_INTERFACE,
        det_trial_id="1",
        det_experiment_id="1",
        det_cluster_id="test_mode",
        trial_seed=config.experiment_seed(),
    )
    workloads = _make_test_workloads(checkpoint_dir.joinpath("checkpoint"), config)
    rendezvous_ports = env.rendezvous_ports()
    rendezvous_info = det.RendezvousInfo(
        addrs=[f"0.0.0.0:{rendezvous_ports[0]}"], addrs2=[f"0.0.0.0:{rendezvous_ports[1]}"], rank=0
    )
    hvd_config = horovod.HorovodContext.from_configs(
        env.experiment_config, rendezvous_info, env.hparams
    )

    return env, workloads, rendezvous_info, hvd_config


def _stop_loading_implementation() -> None:
    raise det.errors.StopLoadingImplementation()


def _init_cluster_mode(
    trial_def: Optional[Type[det.Trial]] = None,
    controller_cls: Optional[Type[det.TrialController]] = None,
    native_context_cls: Optional[Type[det.NativeContext]] = None,
    config: Optional[Dict[str, Any]] = None,
    test: bool = False,
    context_dir: str = "",
    command: Optional[List[str]] = None,
    master_url: Optional[str] = None,
) -> Any:
    if controller_cls is not None and native_context_cls is not None:
        # Case 1: initialize Native implementation.
        if load.RunpyGlobals.is_initialized():
            controller_cls.pre_execute_hook(
                env=load.RunpyGlobals.get_instance().env,
                hvd_config=load.RunpyGlobals.get_instance().hvd_config,
            )
            context = native_context_cls(
                env=load.RunpyGlobals.get_instance().env,
                hvd_config=load.RunpyGlobals.get_instance().hvd_config,
            )
            load.RunpyGlobals.set_runpy_native_result(context, controller_cls)
            context._set_train_fn(_stop_loading_implementation)
            return context

        else:
            _submit_experiment(
                config=config, context_dir=context_dir, command=command, master_url=master_url
            )
            logging.info("Exiting the program after submitting the experiment.")
            sys.exit(0)

    elif trial_def is not None:
        # Case 2: initialize Trial implementation.
        if load.RunpyGlobals.is_initialized():
            load.RunpyGlobals.set_runpy_trial_result(
                trial_def, cast(Type[det.TrialController], trial_def.trial_controller_class)
            )
            _stop_loading_implementation()

        else:
            _submit_experiment(
                config=config,
                test=test,
                context_dir=context_dir,
                command=command,
                master_url=master_url,
            )

    else:
        raise errors.InternalException(
            "Must provide a trial_def if using Trial API or "
            "a controller_cls and a native_context_cls if using Native API."
        )


@contextlib.contextmanager
def _local_execution_manager(new_directory: pathlib.Path) -> Iterator:
    """
    A context manager that temporarily moves the current working directory.

    In trial container, the code always runs from the context directory so the working
    directory is always set to be the context directory.
    In local mode, the context directory might be different from the current working
    directory, which is default to be where the code runs from. We want to unify
    the experience of the user code running in local mode and in trial container. So
    we use this context manager in local mode.
    """

    # TODO(DET-2719): Add context dir to TrainContext and remove this function.
    current_directory = os.getcwd()
    current_path = sys.path.copy()

    try:
        os.chdir(new_directory)

        # Python typically initializes sys.path[0] as the empty string when
        # invoked interactively, which directs Python to search modules in the
        # current directory first. However, this is _not_ happening when this
        # Python function is invoked via the cli. We add it manually here so
        # that _local_trial_from_context and test_one_batch can import the
        # entrypoint by changing the directory to model_def.
        #
        # Reference: https://docs.python.org/3/library/sys.html#sys.path
        sys.path = [""] + sys.path
        yield
    finally:
        os.chdir(current_directory)
        sys.path = current_path


def _local_trial_from_context(
    context_path: pathlib.Path, config: Dict[str, Any], hparams: Dict[str, Any],
) -> det.Trial:
    with _local_execution_manager(context_path):
        checkpoint_dir = tempfile.TemporaryDirectory()

        trial_class = load.load_trial_implementation(config["entrypoint"])
        trial = create_trial_instance(
            trial_class, str(checkpoint_dir), config=config, hparams=hparams
        )

        checkpoint_dir.cleanup()
        return trial


def test_one_batch(
    controller_cls: Optional[Type[det.TrialController]] = None,
    native_context_cls: Optional[Type[det.NativeContext]] = None,
    trial_class: Optional[Type[det.Trial]] = None,
    config: Optional[Dict[str, Any]] = None,
) -> Any:
    # Override the batches_per_step value to 1.
    # TODO(DET-2931): Make the validation step a single batch as well.
    config = {**(config or {}), "batches_per_step": 1}

    logging.info("Running a minimal test experiment locally")
    checkpoint_dir = tempfile.TemporaryDirectory()
    env, workloads, rendezvous_info, hvd_config = _make_local_test_experiment_env(
        checkpoint_dir=pathlib.Path(checkpoint_dir.name), config=config
    )
    logging.info(f"Using hyperparameters: {env.hparams}.")
    logging.debug(f"Using a test experiment config: {env.experiment_config}.")

    if native_context_cls is not None and controller_cls is not None:
        # Case 1: test one batch for Native implementation.
        controller_cls.pre_execute_hook(env=env, hvd_config=hvd_config)
        context = native_context_cls(env=env, hvd_config=hvd_config)

        def train_fn() -> None:
            controller = cast(Type[det.TrialController], controller_cls).from_native(
                context=context,
                env=env,
                workloads=workloads,
                load_path=None,
                rendezvous_info=rendezvous_info,
                hvd_config=hvd_config,
            )
            controller.run()
            checkpoint_dir.cleanup()

        context._set_train_fn(train_fn)
        logging.info(
            "Note: to submit an experiment to the cluster, change local parameter to False"
        )
        return context

    elif trial_class is not None:
        # Case 2: test one batch for Trial implementation.
        controller = load.load_controller_from_trial(
            trial_class=trial_class,
            env=env,
            workloads=workloads,
            load_path=None,
            rendezvous_info=rendezvous_info,
            hvd_config=hvd_config,
        )
        controller.run()
        checkpoint_dir.cleanup()
        logging.info(
            "Note: to submit an experiment to the cluster, change local parameter to False"
        )

    else:
        raise errors.InternalException(
            "Must provide a trial_def if using Trial API or "
            "a controller_cls and a native_context_cls if using Native API."
        )


def init_native(
    trial_def: Optional[Type[det.Trial]] = None,
    controller_cls: Optional[Type[det.TrialController]] = None,
    native_context_cls: Optional[Type[det.NativeContext]] = None,
    config: Optional[Dict[str, Any]] = None,
    local: bool = False,
    test: bool = False,
    context_dir: str = "",
    command: Optional[List[str]] = None,
    master_url: Optional[str] = None,
) -> Any:
    determined_common.set_logger(
        util.debug_mode() or det.ExperimentConfig(config or {}).debug_enabled()
    )

    if local:
        if not test:
            logging.warn("local training is not supported, testing instead")

        with _local_execution_manager(pathlib.Path(context_dir).resolve()):
            return test_one_batch(
                controller_cls=controller_cls,
                native_context_cls=native_context_cls,
                trial_class=trial_def,
                config=config,
            )

    else:
        return _init_cluster_mode(
            trial_def=trial_def,
            controller_cls=controller_cls,
            native_context_cls=native_context_cls,
            config=config,
            test=test,
            context_dir=context_dir,
            command=command,
            master_url=master_url,
        )


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
            A class definition implementing the ``det.Trial`` interface.

        config:
            A dictionary representing the experiment configuration to be
            associated with the experiment.

        local:
            A boolean indicating if training will happen locally. When
            ``False``, the experiment will be submitted to the Determined
            cluster. Defaults to ``False``.

        test:
            A boolean indicating if the experiment should be shortened to a
            minimal loop of training, validation, and checkpointing.
            ``test=True`` is useful quick iterating during model porting or
            debugging because common errors will surface more quickly.
            Defaults to ``False``.

        context_dir:
            A string filepath that defines the context directory. All model
            code will be executed with this as the current working directory.

            When ``local=False``, this argument is required. All files in this
            directory will be uploaded to the Determined cluster. The total
            size of this directory must be under 96 MB.

            When ``local=True``, this argument is optional and assumed to be
            the current working directory by default.

        command:
            A list of strings that is used as the entrypoint of the training
            script in the Determined task environment. When executing this
            function via a python script, this argument is inferred to be
            ``sys.argv`` by default. When executing this function via IPython
            or Jupyter notebook, this argument is required.

            Example: When creating an experiment by running "python train.py
            --flag value", the default command is inferred as ["train.py",
            "--flag", "value"].

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

    return init_native(
        trial_def=trial_def,
        config=config,
        local=local,
        test=test,
        context_dir=context_dir,
        command=command,
        master_url=master_url,
    )


def create_trial_instance(
    trial_def: Type[det.Trial],
    checkpoint_dir: str,
    config: Optional[Dict[str, Any]] = None,
    hparams: Optional[Dict[str, Any]] = None,
) -> det.Trial:
    """
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
    determined_common.set_logger(
        util.debug_mode() or det.ExperimentConfig(config or {}).debug_enabled()
    )
    env, workloads, rendezvous_info, hvd_config = _make_local_test_experiment_env(
        checkpoint_dir=pathlib.Path(checkpoint_dir), config=config, hparams=hparams
    )
    trial_context = trial_def.trial_context_class(env, hvd_config)
    return trial_def(trial_context)
