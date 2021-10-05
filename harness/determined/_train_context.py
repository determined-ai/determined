import abc
import contextlib
import functools
import logging
import shutil
import socket
import tempfile
from typing import Any, Callable, Dict, List, Optional

import determined as det
from determined import constants, horovod, ipc


class TrialContext(metaclass=abc.ABCMeta):
    """
    TrialContext is the system-provided API to a Trial class.
    """

    def __init__(
        self,
        env: det.EnvContext,
        hvd_config: horovod.HorovodContext,
        rendezvous_info: det.RendezvousInfo,
    ) -> None:
        self.env = env
        self.hvd_config = hvd_config
        self.rendezvous_info = rendezvous_info

        if hvd_config.use:
            rank_info = RankInfo(
                rank=horovod.hvd.rank(),
                size=horovod.hvd.size(),
                local_rank=horovod.hvd.local_rank(),
                local_size=horovod.hvd.local_size(),
                cross_rank=horovod.hvd.cross_rank(),
                cross_size=horovod.hvd.cross_size(),
            )
        else:
            rank_info = RankInfo(
                rank=0,
                size=1,
                local_rank=0,
                local_size=1,
                cross_rank=0,
                cross_size=1,
            )

        self.distributed = DistributedContext(
            rank_info=rank_info,
            chief_ip=rendezvous_info.container_addrs[0],
            port_offset=env.det_trial_unique_port_offset,
        )
        self._stop_requested = False

    @classmethod
    def from_config(cls, config: Dict[str, Any]) -> "TrialContext":
        """
        Create an context object suitable for debugging outside of Determined.

        An example for a subclass of :class:`~determined.pytorch._pytorch_trial.PyTorchTrial`:

        .. code-block:: python

            config = { ... }
            context = det.pytorch.PyTorchTrialContext.from_config(config)
            my_trial = MyPyTorchTrial(context)

            train_ds = my_trial.build_training_data_loader()
            for epoch_idx in range(3):
                for batch_idx, batch in enumerate(train_ds):
                    metrics = my_trial.train_batch(batch, epoch_idx, batch_idx)
                    ...

        An example for a subclass of :class:`~determined.keras._tf_keras_trial.TFKerasTrial`:

        .. code-block:: python

            config = { ... }
            context = det.keras.TFKerasTrialContext.from_config(config)
            my_trial = tf_keras_one_var_model.OneVarTrial(context)

            model = my_trial.build_model()
            model.fit(my_trial.build_training_data_loader())
            eval_metrics = model.evaluate(my_trial.build_validation_data_loader())

        Arguments:
            config: An experiment config file, in dictionary form.
        """
        env_context, rendezvous_info, hvd_config = det._make_local_execution_env(
            managed_training=False,
            test_mode=False,
            config=config,
            checkpoint_dir="/tmp",
            limit_gpus=1,
        )
        return cls(env_context, hvd_config, rendezvous_info)

    def get_experiment_config(self) -> Dict[str, Any]:
        """
        Return the experiment configuration.
        """
        return self.env.experiment_config

    def get_data_config(self) -> Dict[str, Any]:
        """
        Return the data configuration.
        """
        return self.get_experiment_config().get("data", {})

    def get_experiment_id(self) -> int:
        """
        Return the experiment ID of the current trial.
        """
        return int(self.env.det_experiment_id)

    def get_global_batch_size(self) -> int:
        """
        Return the global batch size.
        """
        return self.env.global_batch_size

    def get_per_slot_batch_size(self) -> int:
        """
        Return the per-slot batch size. When a model is trained with a single GPU, this is equal to
        the global batch size. When multi-GPU training is used, this is equal to the global batch
        size divided by the number of GPUs used to train the model.
        """
        return self.env.per_slot_batch_size

    def get_trial_id(self) -> int:
        """
        Return the trial ID of the current trial.
        """
        return int(self.env.det_trial_id)

    def get_trial_seed(self) -> int:
        return self.env.trial_seed

    def get_hparams(self) -> Dict[str, Any]:
        """
        Return a dictionary of hyperparameter names to values.
        """
        return self.env.hparams

    def get_hparam(self, name: str) -> Any:
        """
        Return the current value of the hyperparameter with the given name.
        """
        if name not in self.env.hparams:
            raise ValueError(
                "Could not find name '{}' in experiment "
                "hyperparameters. Please check your experiment "
                "configuration 'hyperparameters' section.".format(name)
            )
        if name == "global_batch_size":
            logging.warning(
                "Please use `context.get_per_slot_batch_size()` and "
                "`context.get_global_batch_size()` instead of accessing "
                "`global_batch_size` directly."
            )
        return self.env.hparams[name]

    def get_stop_requested(self) -> bool:
        """
        Return whether a trial stoppage has been requested.
        """
        return self._stop_requested

    def set_stop_requested(self, stop_requested: bool) -> None:
        """
        Set a flag to request a trial stoppage. When this flag is set to True,
        we finish the step, checkpoint, then exit.
        """
        if not isinstance(stop_requested, bool):
            raise AssertionError("stop_requested must be a boolean")

        logging.info(
            "A trial stoppage has requested. The trial will be stopped "
            "at the end of the current step."
        )
        self._stop_requested = stop_requested

    def get_initial_batch(self) -> int:
        return self.env.latest_batch


class RankInfo:
    """
    RankInfo was worker identity information that is:
     - dependent on the launch layer
     - created/used in the worker process
    """

    def __init__(
        self,
        *,
        rank: int,
        size: int,
        local_rank: int,
        local_size: int,
        cross_rank: int,
        cross_size: int,
    ) -> None:
        self._rank = rank
        self._size = size
        self._local_rank = local_rank
        self._local_size = local_size
        self._cross_rank = cross_rank
        self._cross_size = cross_size

    @property
    def rank(self) -> int:
        return self._rank

    @property
    def size(self) -> int:
        return self._size

    @property
    def local_rank(self) -> int:
        return self._local_rank

    @property
    def local_size(self) -> int:
        return self._local_size

    @property
    def cross_rank(self) -> int:
        return self._cross_rank

    @property
    def cross_size(self) -> int:
        return self._cross_size


class DistributedContext:
    """
    DistributedContext  provides useful methods for effective distributed training.
    """

    def __init__(
        self,
        rank_info: RankInfo,
        chief_ip: Optional[str] = None,
        pub_port: int = constants.INTER_TRAIN_PROCESS_COMM_PORT_1,
        pull_port: int = constants.INTER_TRAIN_PROCESS_COMM_PORT_2,
        port_offset: int = 0,
        force_tcp: bool = False,
    ) -> None:
        self._info = rank_info
        self._pub_port = pub_port + port_offset
        self._pull_port = pull_port + port_offset
        self._chief_ip = chief_ip

        self._is_chief = self._info.rank == 0
        self._is_local_chief = self._info.local_rank == 0

        if self._info.cross_size > 1:
            if chief_ip is None:
                raise AssertionError(
                    f"rank_info has cross_size ({self._info.cross_size}) but chief_ip was not "
                    "provided.  When cross_size > 1, the chief_ip parameter is required."
                )
            self._chief_ip = chief_ip
        else:
            # When cross_size == 1, always contact the chief as localhost.
            self._chief_ip = "127.0.0.1"

        self._init_ipc(force_tcp)

    def _init_ipc(self, force_tcp: bool) -> None:
        if self._info.size < 2:
            # No broadcasting necessary.
            return

        # Global broadcast server.
        if self._is_chief:
            logging.debug(f"Chief setting up server with ports {self._pub_port}/{self._pull_port}.")
            self._chief_zmq = ipc.ZMQBroadcastServer(
                num_connections=self._info.size - 1,
                pub_url=f"tcp://*:{self._pub_port}",
                pull_url=f"tcp://*:{self._pull_port}",
            )
            self._chief_zmq.safe_start(lambda: None)

        else:
            logging.debug(
                f"Non-Chief {self._info.rank} setting up comm to "
                f"{self._chief_ip} w/ ports "
                f"{self._pub_port}/{self._pull_port}."
            )
            self._worker_zmq = ipc.ZMQBroadcastClient(
                srv_pub_url=f"tcp://{self._chief_ip}:{self._pub_port}",
                srv_pull_url=f"tcp://{self._chief_ip}:{self._pull_port}",
            )
            self._worker_zmq.safe_start()

        if self._info.local_size < 2:
            # No local broadcasting necessary.
            return

        # Local broadcast server.
        self.tempdir = None
        if self._is_local_chief:
            pub_url = None
            pull_url = None
            if hasattr(socket, "AF_UNIX") and not force_tcp:
                # On systems with unix sockets, we get a slight performance bump by using them.
                self.tempdir = tempfile.mkdtemp(prefix="ipc")
                pub_url = f"ipc://{self.tempdir}/pub.sock"
                pull_url = f"ipc://{self.tempdir}/pull.sock"

            logging.debug(f"Local Chief setting up server with urls {pub_url}/{pull_url}.")
            self._local_chief_zmq = ipc.ZMQBroadcastServer(
                num_connections=self._info.local_size - 1,
                pub_url=pub_url,
                pull_url=pull_url,
            )

            if pub_url is None:
                pub_url = f"tcp://localhost:{self._local_chief_zmq.get_pub_port()}"

            if pull_url is None:
                pull_url = f"tcp://localhost:{self._local_chief_zmq.get_pull_port()}"

            # Do a global allgather to initialize local clients on every node.
            local_chief = (self._info.cross_rank, pub_url, pull_url)
            _ = self._zmq_allgather(local_chief)
            self._local_chief_zmq.safe_start(lambda: None)

        else:
            # Start with the global allgather.
            all_local_chiefs = self._zmq_allgather(None)
            my_local_chief = [
                x for x in all_local_chiefs if x is not None and x[0] == self._info.cross_rank
            ]
            assert len(my_local_chief) == 1, (
                f"did not find exactly 1 local_chief for machine {self._info.cross_rank} "
                f"in {all_local_chiefs}"
            )
            _, pub_url, pull_url = my_local_chief[0]

            assert isinstance(pub_url, str), f"invalid pub_url: {pub_url}"
            assert isinstance(pull_url, str), f"invalid pub_url: {pull_url}"

            logging.debug(f"Local Worker setting up server with urls {pub_url}/{pull_url}.")
            self._local_worker_zmq = ipc.ZMQBroadcastClient(pub_url, pull_url)
            self._local_worker_zmq.safe_start()

    def close(self) -> None:
        # if statements in close() mirror the if statements of _init_ipc().
        if self._info.size < 2:
            return

        # Global broadcast server.
        if self._is_chief:
            self._chief_zmq.close()
        else:
            self._worker_zmq.close()

        if self._info.local_size < 2:
            return

        # Local broadcast server.
        if self._is_local_chief:
            self._local_chief_zmq.close()
            if self.tempdir is not None:
                shutil.rmtree(self.tempdir)
                self.tempdir = None
        else:
            self._local_worker_zmq.close()

    def get_rank(self) -> int:
        """
        Return the rank of the process in the trial. The rank of a process is a
        unique ID within the trial; that is, no two processes in the same trial
        will be assigned the same rank.
        """
        return self._info.rank

    def get_local_rank(self) -> int:
        """
        Return the rank of the process on the agent. The local rank of a process
        is a unique ID within a given agent and trial; that is, no two processes
        in the same trial that are executing on the same agent will be assigned
        the same rank.
        """
        return self._info.local_rank

    def get_size(self) -> int:
        """
        Return the number of slots this trial is running on.
        """
        return self._info.size

    def get_num_agents(self) -> int:
        """
        Return the number of agents this trial is running on.
        """
        return self._info.cross_size

    def _zmq_gather(self, stuff: Any) -> Optional[List]:
        """
        Gather stuff to the chief.  The chief returns a list of all stuff, and workers return None.
        """
        if self._info.size < 2:
            return [stuff]
        logging.debug(f"Worker {self.get_rank()} beginning zmq gather.")
        if self._is_chief:
            worker_stuff, _ = self._chief_zmq.gather_with_polling(lambda: None)
            self._chief_zmq.broadcast(None)
            out = [stuff, *worker_stuff]  # type: Optional[List]
        else:
            self._worker_zmq.send(stuff)
            # Synchronize with the chief so that there is no risk of accidentally calling send()
            # for a future gather before all workers have called send() on this gather.
            _ = self._worker_zmq.recv()
            out = None
        logging.debug(f"Worker {self.get_rank()} finished zmq gather.")
        return out

    def _zmq_gather_local(self, stuff: Any) -> Optional[List]:
        """
        Gather stuff to the local chief.  The local chief returns a list of all stuff, and local
        workers return None.
        """
        if self._info.local_size < 2:
            return [stuff]
        logging.debug(f"Worker {self.get_rank()} beginning zmq gather local.")
        if self._is_local_chief:
            worker_stuff, _ = self._local_chief_zmq.gather_with_polling(lambda: None)
            self._local_chief_zmq.broadcast(None)
            out = [stuff, *worker_stuff]  # type: Optional[List]
        else:
            self._local_worker_zmq.send(stuff)
            # Synchronize with the chief so that there is no risk of accidentally calling send()
            # for a future gather before all workers have called send() on this gather.
            _ = self._local_worker_zmq.recv()
            out = None
        logging.debug(f"Worker {self.get_rank()} finished zmq gather local.")
        return out

    def _zmq_allgather(self, stuff: Any) -> List:
        """
        Gather stuff to the chief and broadcast all of it back to the workers.
        """
        if self._info.size < 2:
            return [stuff]
        logging.debug(f"Worker {self.get_rank()} beginning zmq allgather.")
        if self._is_chief:
            worker_stuff, _ = self._chief_zmq.gather_with_polling(lambda: None)
            all_stuff = [stuff, *worker_stuff]
            self._chief_zmq.broadcast(all_stuff)
        else:
            self._worker_zmq.send(stuff)
            all_stuff = self._worker_zmq.recv()
        logging.debug(f"Worker {self.get_rank()} finished zmq allgather.")
        return all_stuff

    def _zmq_allgather_local(self, stuff: Any) -> List:
        """
        Gather stuff to the local chief and broadcast all of it back to the local workers.
        """
        if self._info.local_size < 2:
            return [stuff]
        logging.debug(f"Worker {self.get_rank()} beginning zmq local allgather.")
        if self._is_local_chief:
            worker_stuff, _ = self._local_chief_zmq.gather_with_polling(lambda: None)
            all_stuff = [stuff, *worker_stuff]
            self._local_chief_zmq.broadcast(all_stuff)
        else:
            self._local_worker_zmq.send(stuff)
            all_stuff = self._local_worker_zmq.recv()
        logging.debug(f"Worker {self.get_rank()} finished zmq local allgather.")
        return all_stuff

    def _zmq_broadcast(self, stuff: Any) -> Any:
        """
        Every worker gets the value sent by the chief.
        """
        if self._info.size < 2:
            return stuff
        if self._is_chief:
            self._chief_zmq.broadcast(stuff)
        else:
            stuff = self._worker_zmq.recv()
        return stuff

    def _zmq_broadcast_local(self, stuff: Any = None) -> Any:
        """
        Every worker gets the value sent by the local chief.
        """
        if self._info.local_size < 2:
            return stuff
        if self._is_local_chief:
            self._local_chief_zmq.broadcast(stuff)
        else:
            stuff = self._local_worker_zmq.recv()
        return stuff

    def _local_chief_contextmanager(self, fn: Callable) -> Callable:
        """
        Wrap a contextmanager such that the real context manager only runs on the chief, but the
        results are distributed to all the local workers.
        """
        if self._is_local_chief:

            @functools.wraps(fn)
            @contextlib.contextmanager
            def _fn(*args: Any, **kwargs: Any) -> Any:
                with fn(*args, **kwargs) as out:
                    # broadcast to local workers
                    _ = self._zmq_broadcast_local(out)
                    try:
                        yield out
                    finally:
                        # wait for local workers
                        _ = self._zmq_gather_local(None)

        else:

            @functools.wraps(fn)
            @contextlib.contextmanager
            def _fn(*__: Any, **___: Any) -> Any:
                # wait for local chief
                out = self._zmq_broadcast_local(None)
                try:
                    yield out
                finally:
                    # wait for local workers
                    _ = self._zmq_gather_local(None)

        return _fn
