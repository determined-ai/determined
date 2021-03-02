import logging
from typing import Any, List, Optional, cast

import determined as det
from determined import constants, ipc
from determined.horovod import hvd
from determined_common import check


class MetricsReduceHelper:
    def __init__(self, context: det.TrialContext) -> None:
        self.context = context
        self.env = self.context.env
        self.hvd_config = self.context.hvd_config
        self.rendezvous_info = self.context.rendezvous_info

        self.batch_size = self.context.get_per_slot_batch_size()
        self.scheduling_unit = self.env.experiment_config.scheduling_unit()

        logging.debug("Starting MetricsReducer initialization.")

        self.is_chief = self.context.distributed.is_chief()
        training_process_rank = self.context.distributed.get_local_rank()

        logging.debug(
            f"Training coordination initialized on local rank {training_process_rank}, "
            f"using hvd: {self.hvd_config.use}."
        )

        # Initialize communication directly between training processes.
        self.train_process_comm_chief = None  # type: Optional[ipc.ZMQBroadcastServer]
        self.train_process_comm_worker = None  # type: Optional[ipc.ZMQBroadcastClient]
        if self.hvd_config.use:
            self._initialize_train_process_comm()

    def _initialize_train_process_comm(self) -> None:
        check.true(self.hvd_config.use)

        srv_pub_port = (
            constants.INTER_TRAIN_PROCESS_COMM_PORT_1 + self.env.det_trial_unique_port_offset
        )
        srv_pull_port = (
            constants.INTER_TRAIN_PROCESS_COMM_PORT_2 + self.env.det_trial_unique_port_offset
        )

        if self.is_chief:
            logging.debug(f"Chief setting up server with ports {srv_pub_port}/{srv_pull_port}.")
            self.train_process_comm_chief = ipc.ZMQBroadcastServer(
                num_connections=self.env.experiment_config.slots_per_trial() - 1,
                pub_port=srv_pub_port,
                pull_port=srv_pull_port,
            )
        else:
            chief_ip_address = self.rendezvous_info.get_ip_addresses()[0]
            logging.debug(
                f"Non-Chief {hvd.rank()} setting up comm to "
                f"{chief_ip_address} w/ ports "
                f"{srv_pub_port}/{srv_pull_port}."
            )
            self.train_process_comm_worker = ipc.ZMQBroadcastClient(
                srv_pub_url=f"tcp://{chief_ip_address}:{srv_pub_port}",
                srv_pull_url=f"tcp://{chief_ip_address}:{srv_pull_port}",
            )

    def global_barrier(self) -> None:
        # Executes a barrier by communicating directly between worker processes via ZMQ.
        logging.debug(f"Worker {self.context.distributed.get_rank()} entering global barrier.")
        if self.is_chief:
            self.train_process_comm_chief = cast(
                ipc.ZMQBroadcastServer, self.train_process_comm_chief
            )
            self.train_process_comm_chief.gather_with_polling(lambda: None)
            self.train_process_comm_chief.broadcast(None)
        else:
            self.train_process_comm_worker = cast(
                ipc.ZMQBroadcastClient, self.train_process_comm_worker
            )
            self.train_process_comm_worker.send([None])
            # Synchronize with the chief so that there is no risk of accidentally calling send()
            # for a future gather before all workers have called send() on this gather.
            _ = self.train_process_comm_worker.recv()
        logging.debug(f"Worker {self.context.distributed.get_rank()} exiting global barrier.")

    def allgather_metrics(self, metrics: Any) -> List:
        if not self.hvd_config.use:
            return [metrics]

        if self.is_chief:
            self.train_process_comm_chief = cast(
                ipc.ZMQBroadcastServer, self.train_process_comm_chief
            )
            logging.debug(f"Chief {hvd.rank()} beginning allgathering metrics.")
            worker_stuff, _ = self.train_process_comm_chief.gather_with_polling(lambda: None)
            logging.debug(f"Chief {hvd.rank()} done allgathering metrics.")
            all_metrics = [metrics, *worker_stuff]
            self.train_process_comm_chief.broadcast(all_metrics)
            return all_metrics
        else:
            self.train_process_comm_worker = cast(
                ipc.ZMQBroadcastClient, self.train_process_comm_worker
            )
            logging.debug(f"Worker {hvd.rank()} allgathering metrics.")
            self.train_process_comm_worker.send(metrics)
            return self.train_process_comm_worker.recv()  # type: ignore
