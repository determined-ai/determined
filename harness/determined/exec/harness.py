import faulthandler
import logging
import sys

import determined as det
from determined import _generic, horovod, load
from determined.common import experimental
from determined.common.api import certs


def config_logging(debug: bool) -> None:
    log_level = logging.DEBUG if debug else logging.INFO
    logging.basicConfig(
        level=log_level, format="%(asctime)s:%(levelname)s [%(process)s]: %(message)s"
    )
    logging.getLogger().setLevel(log_level)
    logging.debug("Starting harness.")


def main() -> int:
    info = det.get_cluster_info()
    assert info is not None, "must be run on-cluster"
    assert info.task_type == "TRIAL", f'must be run with task_type="TRIAL", not "{info.task_type}"'

    # TODO: refactor websocket, data_layer, and profiling to to not use the cli_cert.
    certs.cli_cert = certs.default_load(info.master_url)

    # TODO: Don't include EnvContext object in the future high-level APIs for PyTorch or Keras.
    # It was natural to create this big-blob-of-config object, but it was a mistake to pass it into
    # the lowest layers of the harness code; it's too large of an object to be easily mockable,
    # which is part of why building local training mode has always been a challenge.
    #
    # A better pattern is to pass in exactly the information that is necessary at each layer.  We
    # will use that pattern for the future high-level APIs, but it's not worth refactoring e.g. the
    # TFKerasTrialController or EstimatorTrialController to add that functionality, so for now we
    # continue with the legay strategy.

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

    if env.experiment_config.debug_enabled():
        faulthandler.dump_traceback_later(30, repeat=True)

    with det._catch_sys_exit():
        try:
            # TODO: reorder object lifetimes so that the DistributedContext and the GenericContext
            # are created here.  Nothing else in the TrialController or Trial code needs the
            # rendezvous info.  Then we can remove the rendezvous info as an arg from the whole rest
            # of the harness.
            rendezvous_info = info._rendezvous_info
            assert rendezvous_info is not None
            controller = load.prepare_controller(
                env,
                rendezvous_info,
                hvd_config,
            )
        except det.InvalidHP:
            # build a Training API object just to call report_early_exit().
            session = experimental.Session(None, None, None, certs.cli_cert)
            training = _generic.Training(
                session,
                info.trial.trial_id,
                env.trial_run_id,
                info.trial.experiment_id,
                None,
                None,
            )
            training.report_early_exit(_generic.EarlyExitReason.INVALID_HP)
            logging.info("InvalidHP detected during Trial init, worker is exiting")
            return 0

        try:
            controller.run()
        finally:
            # TODO: Refactor load_trial so that it takes a generic context as input.
            # That way we can just be inside a context manager and we don't have to keep track of
            # errors so closely.
            controller.context.distributed.close()

            # Don't hang with debug=true.
            if env.experiment_config.debug_enabled():
                faulthandler.cancel_dump_traceback_later()

    return 0


if __name__ == "__main__":
    sys.exit(main())
