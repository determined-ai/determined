import faulthandler
import logging
import os
import pathlib
import sys

import determined as det
from determined import _generic, horovod, layers, load
from determined.common import experimental
from determined.common.api import certs


def config_logging(debug: bool) -> None:
    log_level = logging.DEBUG if debug else logging.INFO
    logging.basicConfig(
        level=log_level, format="%(asctime)s:%(levelname)s [%(process)s]: %(message)s"
    )
    logging.getLogger().setLevel(log_level)
    logging.debug("Starting harness.")


def main(
    env: det.EnvContext, rendezvous_info: det.RendezvousInfo, hvd_config: horovod.HorovodContext
) -> int:
    config_logging(env.debug)

    if env.experiment_config.debug_enabled():
        faulthandler.dump_traceback_later(30, repeat=True)

    with det._catch_sys_exit():
        try:
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
                int(env.det_trial_id),
                env.trial_run_id,
                int(env.det_experiment_id),
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

    return 0


if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("worker_process_env_path must be provided as a commandline argument", file=sys.stderr)
        sys.exit(1)

    # Load the worker process env.
    wpc = layers.WorkerProcessContext.from_file(pathlib.Path(sys.argv[1]))

    # API code expects credential to be available as an environment variable
    os.environ["DET_ALLOCATION_SESSION_TOKEN"] = wpc.env.det_allocation_token

    # TODO: refactor websocket, data_layer, and profiling to to not use the cli_cert.
    master_url = (
        f"http{'s' if wpc.env.use_tls else ''}://" f"{wpc.env.master_addr}:{wpc.env.master_port}"
    )
    certs.cli_cert = certs.default_load(master_url=master_url)

    sys.exit(main(wpc.env, wpc.rendezvous_info, wpc.hvd_config))
