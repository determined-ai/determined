import argparse
import contextlib
import faulthandler
import logging
import sys
from typing import Iterator, Optional

import determined as det
from determined import _generic, horovod, load
from determined.common.api import certs


def config_logging(debug: bool) -> None:
    log_level = logging.DEBUG if debug else logging.INFO
    logging.basicConfig(
        level=log_level, format="%(asctime)s:%(levelname)s [%(process)s]: %(message)s"
    )
    logging.getLogger().setLevel(log_level)
    logging.debug("Starting harness.")


@contextlib.contextmanager
def maybe_periodic_stacktraces(debug_enabled: bool) -> Iterator[None]:
    if debug_enabled:
        faulthandler.dump_traceback_later(30, repeat=True)
    try:
        yield
    finally:
        if debug_enabled:
            faulthandler.cancel_dump_traceback_later()


def main(chief_ip: Optional[str]) -> int:
    info = det.get_cluster_info()
    assert info is not None, "must be run on-cluster"
    assert info.task_type == "TRIAL", f'must be run with task_type="TRIAL", not "{info.task_type}"'

    # TODO: refactor data_layer, and profiling to to not use the cli_cert.
    certs.cli_cert = certs.default_load(info.master_url)

    # TODO: Don't include EnvContext object in the future high-level APIs for PyTorch or Keras.
    # It was natural to create this big-blob-of-config object, but it was a mistake to pass it into
    # the lowest layers of the harness code; it's too large of an object to be easily mockable,
    # which is part of why building local training mode has always been a challenge.
    #
    # A better pattern is to pass in exactly the information that is necessary at each layer.  We
    # will use that pattern for the future high-level APIs, but it's not worth refactoring e.g. the
    # TFKerasTrialController or EstimatorTrialController to add that functionality, so for now we
    # continue with the legacy strategy.

    env = det.EnvContext(
        master_url=info.master_url,
        master_cert_file=info.master_cert_file,
        master_cert_name=info.master_cert_name,
        experiment_config=info.trial._config,
        container_id=info.container_id,
        hparams=info.trial.hparams,
        latest_checkpoint=info.latest_checkpoint,
        latest_batch=info.trial._latest_batch,
        use_gpu=bool(info.gpu_uuids),
        container_gpus=info.gpu_uuids,
        slot_ids=info.slot_ids,
        debug=info.trial._debug,
        det_trial_unique_port_offset=info.trial._unique_port_offset,
        det_trial_id=str(info.trial.trial_id),
        det_experiment_id=str(info.trial.experiment_id),
        det_agent_id=info.agent_id,
        det_cluster_id=info.cluster_id,
        trial_seed=info.trial.trial_seed,
        trial_run_id=info.trial._trial_run_id,
        allocation_id=info.allocation_id,
        managed_training=True,
        test_mode=False,
        on_cluster=True,
    )

    multi_machine_trial = len(info.container_addrs) > 1
    hvd_config = horovod.HorovodContext.from_configs(
        env.experiment_config, env.hparams, multi_machine_trial
    )

    config_logging(env.debug)

    with maybe_periodic_stacktraces(env.debug):
        # Step 1: Load user code.
        # We can't build a generic.Context until we have a RankInfo, and we can't build a RankInfo
        # without horovod, and we can't load the right horovod until we know which Trial class the
        # user implemented.
        trial_class, controller_class = load.get_trial_and_controller_class(env.experiment_config)

        # Step 2: Initialize framework-specific details (horovod, random seeds, etc).
        controller_class.pre_execute_hook(env, hvd_config)

        # Step 3: Now that horovod is initialized, we can build a RankInfo object.
        # It is always expected that the training code can figure this out based on how the
        # launch layer launched the code.
        if hvd_config.use:
            distributed = _generic.DistributedContext(
                rank=horovod.hvd.rank(),
                size=horovod.hvd.size(),
                local_rank=horovod.hvd.local_rank(),
                local_size=horovod.hvd.local_size(),
                cross_rank=horovod.hvd.cross_rank(),
                cross_size=horovod.hvd.cross_size(),
                chief_ip=chief_ip,
                port_offset=info.task_type == "TRIAL" and info.trial._unique_port_offset or 0,
            )
        else:
            distributed = _generic.DummyDistributed()

        # Step 4: Let generic.init() create the generic.Context.
        with _generic.init(distributed=distributed) as generic_context:
            trial_context = trial_class.trial_context_class(generic_context, env, hvd_config)

            # Step 5: Instantiate the user's Trial.
            trial_inst = trial_class(trial_context)

            # Step 6: Create a TrialController and execute training
            logging.info(f"Creating {controller_class.__name__} with {trial_class.__name__}.")
            controller = controller_class.from_trial(
                trial_inst=trial_inst,
                context=trial_context,
                env=env,
                hvd_config=hvd_config,
            )

            controller.run()

    return 0


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--chief-ip")
    args = parser.parse_args()
    sys.exit(main(args.chief_ip))
