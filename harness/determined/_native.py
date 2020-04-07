import enum
import math
import pathlib
import random
import sys
import tempfile
import zipfile
from typing import Any, Dict, List, Optional, Tuple, Type, cast

import determined as det
import determined_common.api.authentication as auth
from determined import constants, errors, gpu, horovod, load, workload
from determined_common import api, check, context, util


class Mode(enum.Enum):
    SUBMIT = "submit"
    TEST = "test"


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


def _get_current_filepath() -> pathlib.Path:
    return pathlib.Path(sys.argv[0]).resolve()


def _get_current_args() -> List:
    return sys.argv[1:]


def set_command_default(
    context_dir: pathlib.Path, command: Optional[List[str]] = None
) -> List[str]:
    if not command or len(command) == 0:
        if not _in_ipython():
            exp_path = _get_current_filepath()
            exp_rel_path = exp_path.relative_to(context_dir)
            command = [str(exp_rel_path), *_get_current_args()]
        else:
            command = []

        if not (len(command) > 0 and (command[0].endswith(".py") or command[0].endswith(".ipynb"))):
            raise errors.InvalidExperimentException(
                "Must specify the command to run the experiment file. "
                "The experiment file needs to have a suffix of .py or .ipynb."
            )
    return command


def make_native_config(config: Optional[Dict[str, Any]], command: List[str]) -> Dict[str, Any]:
    conf = constants.DEFAULT_EXP_CFG.copy()  # type: Dict[str, Any]
    if config is not None:
        conf.update(config)
    conf.setdefault("internal", {})
    conf["internal"]["native"] = {"command": command}
    return conf


def set_native_experiment_defaults(
    config: Optional[Dict[str, Any]], context_dir: str, command: Optional[List[str]] = None,
) -> Tuple[Dict[str, Any], pathlib.Path, List[str]]:
    if context_dir == "":
        raise errors.InvalidExperimentException("Cannot specify the context directory to be empty.")
    exp_context_dir = pathlib.Path(context_dir)
    command = set_command_default(exp_context_dir, command)
    config = make_native_config(config, command)
    return config, exp_context_dir, command


def create_experiment(
    config: Optional[Dict[str, Any]],
    context_dir: str,
    command: Optional[List[str]],
    test_mode: bool = False,
    master_url: Optional[str] = None,
) -> Optional[int]:
    """Create an experiment in a Determined master.

    Args:
        name (Optional[str]): The URL of the Determined master node. If None
        (default), then the master address will be inferred from the
        environment.

    Returns:
        The ID of the created experiment.
    """

    config, exp_context_dir, command = set_native_experiment_defaults(config, context_dir, command)
    print("Creating an experiment with config: {}".format(config))

    if master_url is None:
        master_url = util.get_default_master_address()

    exp_context = context.Context.from_local(exp_context_dir)

    # When a requested_user isn't specified to initialize_session(), the
    # authentication module will attempt to use the token store to grab the
    # current logged-in user. If there is no logged in user found, it will
    # default to constants.DEFAULT_DETERMINED_USER.
    auth.initialize_session(master_url, requested_user=None, try_reauth=True)

    if test_mode:
        exp_id = api.create_test_experiment(master_url, config, exp_context)
    else:
        exp_id = api.create_experiment(master_url, config, exp_context)
    print("Created experiment {}".format(exp_id))

    return exp_id


def get_gpus() -> Tuple[bool, List[str], List[str]]:
    gpu_ids, gpu_uuids = gpu.get_gpu_ids_and_uuids()
    use_gpu = len(gpu_uuids) > 0
    return use_gpu, gpu_uuids, gpu_ids


def generate_test_hparam_values(config: Dict[str, Any]) -> Dict[str, Any]:
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


def make_test_workloads(
    checkpoint_dir: pathlib.Path, config: det.ExperimentConfig
) -> workload.Stream:
    print("Start training a test experiment.")
    interceptor = workload.WorkloadResponseInterceptor()

    print("Training 1 step.")
    yield from interceptor.send(workload.train_workload(1), [config.batches_per_step()])
    metrics = interceptor.metrics_result()
    batch_metrics = metrics["batch_metrics"]
    check.eq(len(batch_metrics), config.batches_per_step())
    print(f"Finished training. Metrics: {batch_metrics}")

    print("Validating.")
    yield from interceptor.send(workload.validation_workload(1), [])
    validation = interceptor.metrics_result()
    v_metrics = validation["validation_metrics"]
    print(f"Finished validating. Validation metrics: {v_metrics}")

    print(f"Saving a checkpoint to {checkpoint_dir}")
    yield workload.checkpoint_workload(), [checkpoint_dir], workload.ignore_workload_response
    print(f"Finished saving a checkpoint to {checkpoint_dir}.")

    yield workload.terminate_workload(), [], workload.ignore_workload_response
    print("The test experiment passed.")


def make_test_experiment_env(
    tmp_dir: pathlib.Path,
    config: Optional[Dict[str, Any]],
    context_dir: str,
    command: Optional[List[str]] = None,
) -> Tuple[det.EnvContext, workload.Stream, det.RendezvousInfo, horovod.HorovodContext]:
    config, exp_context_dir, command = set_native_experiment_defaults(config, context_dir, command)

    # Here wraps the model file into a zip because PyTorchTrial needs a wrapped mode zip file
    # during saving checkpoints.
    zip_path = tmp_dir.joinpath("model_def.zip")
    with zipfile.ZipFile(zip_path, "w") as zf:
        zf.write(command[0], arcname="model_def/__init__.py")

    config_test = det.ExperimentConfig(api.make_test_experiment_config(config))
    hparams = generate_test_hparam_values(config_test)
    use_gpu, container_gpus, slot_ids = get_gpus()
    local_rendezvous_ports = (
        f"{constants.LOCAL_RENDEZVOUS_PORT},{constants.LOCAL_RENDEZVOUS_PORT+1}"
    )

    env = det.EnvContext(
        master_addr="",
        master_port=1,
        container_id="test_mode",
        experiment_config=config_test,
        hparams=hparams,
        initial_workload=workload.train_workload(1, 1, 1),
        latest_checkpoint=None,
        use_gpu=use_gpu,
        container_gpus=container_gpus,
        slot_ids=slot_ids,
        debug=config_test.debug_enabled(),
        workload_manager_type="",
        det_rendezvous_ports=local_rendezvous_ports,
        det_trial_runner_network_interface=constants.AUTO_DETECT_TRIAL_RUNNER_NETWORK_INTERFACE,
        det_trial_id="1",
        det_experiment_id="1",
        det_cluster_id="test_mode",
        trial_seed=config_test.experiment_seed(),
    )
    workloads = make_test_workloads(tmp_dir.joinpath("checkpoint"), config_test)
    rendezvous_ports = env.rendezvous_ports()
    rendezvous_info = det.RendezvousInfo(
        addrs=[f"0.0.0.0:{rendezvous_ports[0]}"], addrs2=[f"0.0.0.0:{rendezvous_ports[1]}"], rank=0
    )
    hvd_config = horovod.HorovodContext.from_configs(
        env.experiment_config, rendezvous_info, env.hparams
    )

    return env, workloads, rendezvous_info, hvd_config


def create(
    trial_def: Type[det.Trial],
    config: Optional[Dict[str, Any]] = None,
    mode: Mode = Mode.SUBMIT,
    context_dir: str = "",
    command: Optional[List[str]] = None,
    master_url: Optional[str] = None,
) -> None:
    if Mode(mode) == Mode.SUBMIT:
        if load.RunpyGlobals.is_initialized():
            load.RunpyGlobals.set_runpy_trial_result(
                trial_def, cast(Type[det.TrialController], trial_def.trial_controller_class),
            )

        else:
            create_experiment(
                config=config, context_dir=context_dir, command=command, master_url=master_url,
            )

    elif Mode(mode) == Mode.TEST:
        print("Running test mode locally.")
        tmp_dir = tempfile.TemporaryDirectory()
        env, workloads, rendezvous_info, hvd_config = make_test_experiment_env(
            tmp_dir=pathlib.Path(tmp_dir.name),
            config=config,
            context_dir=context_dir,
            command=command,
        )
        print(
            "Starting a test experiment.\n"
            f"Using a modified test config: {env.experiment_config}.\n"
            f"Using a set of random hyperparameter values: {env.hparams}."
        )
        controller = load.load_controller_from_trial(
            trial_class=trial_def,
            env=env,
            workloads=workloads,
            load_path=None,
            rendezvous_info=rendezvous_info,
            hvd_config=hvd_config,
        )
        controller.run()
        tmp_dir.cleanup()
        print("Note: to submit a real experiment to the cluster, change mode argument to 'submit'")

    else:
        raise errors.InvalidExperimentException("Must use either test mode or submit mode.")


def init_native(
    controller_cls: Type[det.TrialController],
    native_context_cls: Type[det.NativeContext],
    config: Optional[Dict[str, Any]] = None,
    mode: Mode = Mode.SUBMIT,
    context_dir: str = "",
    command: Optional[List[str]] = None,
    master_url: Optional[str] = None,
) -> Any:
    if Mode(mode) == Mode.SUBMIT:
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
            return context

        else:
            create_experiment(
                config=config, context_dir=context_dir, command=command, master_url=master_url,
            )
            print("Exiting the program after submitting the experiment.")
            sys.exit(0)

    elif Mode(mode) == Mode.TEST:
        print("Running test mode locally.")
        tmp_dir = tempfile.TemporaryDirectory()
        env, workloads, rendezvous_info, hvd_config = make_test_experiment_env(
            tmp_dir=pathlib.Path(tmp_dir.name),
            config=config,
            context_dir=context_dir,
            command=command,
        )
        print(
            "Starting a test experiment.\n"
            f"Using a modified test config: {env.experiment_config}.\n"
            f"Using a set of random hyperparameter values: {env.hparams}."
        )
        controller_cls.pre_execute_hook(
            env=env, hvd_config=hvd_config,
        )
        context = native_context_cls(env=env, hvd_config=hvd_config)

        def train_fn() -> None:
            controller = controller_cls.from_native(
                context=context,
                env=env,
                workloads=workloads,
                load_path=None,
                rendezvous_info=rendezvous_info,
                hvd_config=hvd_config,
            )
            controller.run()
            tmp_dir.cleanup()
            print(
                "Note: to submit a real experiment to the cluster, change mode argument to 'submit'"
            )

        context.set_train_fn(train_fn)
        return context

    else:
        raise errors.InvalidExperimentException("Must use either test mode or submit mode.")
