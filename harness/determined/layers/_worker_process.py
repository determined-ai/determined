import logging
import os
import pathlib
import pickle
import subprocess
import sys
import time
from typing import Any, Dict, List, Optional, cast

import psutil

import determined as det
from determined import constants, horovod, ipc, workload
from determined.common import check


class WorkerProcessContext:
    def __init__(
        self,
        broadcast_pub_port: int,
        broadcast_pull_port: int,
        debug: bool,
        hvd_config: horovod.HorovodContext,
        rendezvous_info: det.RendezvousInfo,
        env: det.EnvContext,
    ) -> None:
        self.broadcast_pub_port = broadcast_pub_port
        self.broadcast_pull_port = broadcast_pull_port
        self.debug = debug
        self.hvd_config = hvd_config
        self.rendezvous_info = rendezvous_info
        self.env = env

    @staticmethod
    def from_file(path: pathlib.Path) -> "WorkerProcessContext":
        with path.open(mode="rb") as f:
            obj = pickle.load(f)
        check.is_instance(obj, WorkerProcessContext, "did not find WorkerProcessContext in file")
        return cast(WorkerProcessContext, obj)

    def to_file(self, path: pathlib.Path) -> None:
        with path.open(mode="wb") as f:
            pickle.dump(self, f)


class SubprocessReceiver:
    """
    SubprocessReceiver is a lightweight wrapper around the ZMQBroadcastClient. ZMQ details are
    handled automatically, while any received workloads are passed along blindly, resulting in a
    network-transparent WorkloadIterator.
    """

    def __init__(self, broadcast_client: ipc.ZMQBroadcastClient):
        self._broadcast_client = broadcast_client

        # Signal to the SubprocessLauncher that the subprocess has started and
        # send the process id so that the SubprocessLauncher can perform health
        # checks on it.
        self._broadcast_client.send(ipc.ConnectedMessage(process_id=os.getpid()))

        # That's literally all we do with the broadcast client.


class SubprocessLauncher:
    def __init__(
        self,
        env: det.EnvContext,
        rendezvous_info: det.RendezvousInfo,
        hvd_config: horovod.HorovodContext,
        python_subprocess_entrypoint: Optional[str] = None,
    ) -> None:

        self.env = env
        self.rendezvous_info = rendezvous_info
        self.hvd_config = hvd_config
        self._python_subprocess_entrypoint = python_subprocess_entrypoint

        self.debug = self.env.experiment_config.debug_enabled()

        # The process ids for the workers that are launched by Horovod. These are different
        # from the main horovod process and sshd processes.
        self._worker_process_ids = []  # type: List[int]

        # Horovod will have a separate training process for each slot.
        self.num_proc = len(self.env.slot_ids) if self.hvd_config.use else 1

        # Step 1: Establish the server for communicating with the subprocess.
        self.broadcast_server = ipc.ZMQBroadcastServer(num_connections=self.num_proc)

        # Step 2: Configure the per-machine WorkerProcessContext.
        self._init_worker_process_env()

        self.is_chief_machine = self.rendezvous_info.get_rank() == 0
        chief_addr = self.rendezvous_info.get_ip_addresses()[0]
        chief_port = self.rendezvous_info.get_ports()[0]

        if self.is_chief_machine:
            # Step 3 (chief): Wait for any peer machines to launch sshd, then launch horovodrun.
            if self.rendezvous_info.get_size() > 1:
                with ipc.ZMQServer(ports=[chief_port], num_connections=1) as server:
                    num_peers = self.rendezvous_info.get_size() - 1
                    responses = server.barrier(num_connections=num_peers, timeout=20)
                    if len(responses) < num_peers:
                        raise AssertionError(
                            f"Chief received sshd ready signal only from {len(responses)} "
                            f"of {num_peers} machines."
                        )
                    logging.debug("Chief finished sshd barrier.")

            if self.hvd_config.use:
                self._subproc = self._launch_horovodrun()
            else:
                self._subproc = self._launch_python_subprocess()

        else:
            # Step 3 (non-chief): launch sshd, wait for it to come up, then signal to the chief.
            self._subproc = self._launch_sshd()

            self._wait_for_sshd_to_start()

            with ipc.ZMQClient(chief_addr, chief_port) as client:
                client.barrier()

    def _init_worker_process_env(self) -> None:
        """
        Initialize the environment variables for the training process.

        TODO(DET-1330): Serialize all environment variables used by training process.
        """

        worker_process_env = WorkerProcessContext(
            broadcast_pub_port=self.broadcast_server.get_pub_port(),
            broadcast_pull_port=self.broadcast_server.get_pull_port(),
            debug=self.debug,
            hvd_config=self.hvd_config,
            rendezvous_info=self.rendezvous_info,
            env=self.env,
        )
        self._worker_process_env_path = pathlib.Path(
            "{}-{}-{}".format(
                constants.TRAIN_PROCESS_ENVIRONMENT_VARIABLE_PATH,
                self.env.det_experiment_id,
                self.env.det_trial_id,
            )
        )
        worker_process_env.to_file(self._worker_process_env_path)

    def _launch_horovodrun(self) -> subprocess.Popen:
        check.true(self.hvd_config.use)
        logging.debug(f"Starting training process on: {self.rendezvous_info.get_rank()}.")

        horovod_process_cmd = horovod.create_run_command(
            num_proc_per_machine=self.num_proc,
            ip_addresses=self.rendezvous_info.get_ip_addresses(),
            env=self.env,
            debug=self.env.experiment_config.debug_enabled(),
            optional_args=self.env.experiment_config.horovod_optional_args(),
            worker_process_env_path=self._worker_process_env_path,
        )
        subprocess_env = {
            **os.environ,
            "NCCL_DEBUG": "INFO",
            "DET_HOROVOD_GLOO_RENDEZVOUS_PORT": str(
                constants.HOROVOD_GLOO_RENDEZVOUS_PORT + self.env.det_trial_unique_port_offset
            ),
        }
        return subprocess.Popen(horovod_process_cmd, env=subprocess_env)

    def _launch_sshd(self) -> subprocess.Popen:
        run_sshd_command = [
            "/usr/sbin/sshd",
            "-p",
            str(constants.HOROVOD_SSH_PORT),
            "-f",
            "/run/determined/ssh/sshd_config",
            "-D",
        ]
        logging.debug(
            f"Non-chief [{self.rendezvous_info.get_rank()}] training process launch "
            f"command: {run_sshd_command}."
        )
        return subprocess.Popen(run_sshd_command)

    def _wait_for_sshd_to_start(self) -> None:
        connection_attempts = 0
        logging.debug(f"Non-chief [{self.rendezvous_info.get_rank()}] waiting for sshd service.")
        while True:
            ssh_attempt_cmd = ["ssh", "localhost", "-p", str(constants.HOROVOD_SSH_PORT), "ls"]
            ssh_attempt_process = subprocess.run(
                ssh_attempt_cmd, timeout=10
            )
            if ssh_attempt_process.returncode == 0:
                logging.debug(
                    f"Non-chief [{self.rendezvous_info.get_rank()}] successfully "
                    "started sshd service."
                )
                break

            # Check that training subprocess is still alive and well.
            self._health_check()

            connection_attempts += 1
            if connection_attempts == 10:
                raise AssertionError("Training process failed to start sshd.")

            logging.info("Waiting for training process to start sshd ...")
            time.sleep(1)

    def _launch_python_subprocess(self) -> subprocess.Popen:
        """
        Call training process without using horovodrun. Only used internally when testing.
        """

        check.is_not_none(self._python_subprocess_entrypoint)
        self._python_subprocess_entrypoint = cast(str, self._python_subprocess_entrypoint)

        # Construct the command to launch the non-horovod training subprocess.
        python = sys.executable
        python_cmd = [
            python,
            "-m",
            self._python_subprocess_entrypoint,
            str(self._worker_process_env_path),
        ]
        return subprocess.Popen(python_cmd)

    def _do_startup_message_sequence(self) -> None:
        # Wait for a ConnectedMessage from every worker.
        responses, exception_received = self.broadcast_server.gather_with_polling(
            self._health_check
        )

        if exception_received:
            raise det.errors.WorkerError("Training process died.")

        for response in responses:
            check.is_instance(
                response,
                ipc.ConnectedMessage,
                f"Did not receive ConnectedMessage from worker. Got: {response}",
            )
            response = cast(ipc.ConnectedMessage, response)
            self._worker_process_ids.append(response.process_id)

    def run(self) -> None:
        """
        The main control loop for controlling worker processes.
        """

        try:
            self._do_startup_message_sequence()

            if self.rendezvous_info.get_rank() == 0:
                # The chief node just waits for horovodrun to exit.
                ret = self._subproc.wait()
                if ret != 0:
                    raise ValueError(f"horovodrun exited with error code: {ret}")
            else:
                # The worker nodes wait for for all ssh sessions to exit, then kills sshd.
                while self._worker_process_ids:
                    time.sleep(1)
                    if self._subproc.poll() is not None:
                        raise det.errors.WorkerError("sshd process died")
                    for pid in self._worker_process_ids:
                        if not psutil.pid_exists(pid):
                            logging.debug("detected that sshd session died")
                            self._worker_process_ids.remove(pid)

                # Wait a few seconds, in case some processes are in the process of exiting but have
                # not finished logging quite yet.
                time.sleep(3)

                # Shut down sshd.
                self._subproc.kill()
                self._subproc.wait()

        finally:
            self.broadcast_server.close()

            # If we wait for delete to be called when the interpreter shutdowns, we sometimes get a
            # (harmless) stack trace from delete a socket with a `None` value. This is due to race
            # conditions between when the interpreter deletes the weakref module and when pyzmq
            # calls the weakref module in a __del__ method. We work around this by triggering the
            # garbage collection earlier.
            del self.broadcast_server

    def _health_check(self) -> None:
        """
        Raise an error if the main subprocess dies.
        """

        if self._subproc.poll() is not None:
            raise det.errors.WorkerError("Training process died.")

        for subprocess_id in self._worker_process_ids:
            if not psutil.pid_exists(subprocess_id):
                # Wait a few seconds, in case some processes are in the process of exiting but have
                # not finished logging quite yet.
                time.sleep(3)
                raise det.errors.WorkerError("Detected that worker process died.")
