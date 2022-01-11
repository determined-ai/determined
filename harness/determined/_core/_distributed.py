import contextlib
import functools
import logging
import shutil
import socket
import tempfile
from typing import Any, Callable, List, Optional

from determined import constants, ipc


class DistributedContext:
    """
    DistributedContext provides useful methods for effective distributed training.

    A DistributedContext has the following required args:
     - rank: the index of this worker in the entire job
     - size: the number of workers in the entire job
     - local_rank: the index of this worker on this machine
     - local_size: the number of workers on this machine
     - cross_rank: the index of this machine in the entire job
     - cross_size: the number of this machines in the entire job

    Additionally, any time that cross_size > 0, you must also provide:
     - chief_ip: the ip address to reach the chief worker (where rank==0)
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
        chief_ip: Optional[str] = None,
        pub_port: int = constants.INTER_TRAIN_PROCESS_COMM_PORT_1,
        pull_port: int = constants.INTER_TRAIN_PROCESS_COMM_PORT_2,
        port_offset: int = 0,
        force_tcp: bool = False,
    ) -> None:
        rank_args = (rank, size, local_rank, local_size, cross_rank, cross_size)
        if sum(x is not None for x in rank_args) not in (0, 6):
            raise ValueError(
                "rank, size, local_rank, local_size, cross_rank, and cross_size must all be "
                "provided if any are provided"
            )

        self.rank = rank if rank is not None else 0
        self.size = size if size is not None else 1
        self.local_rank = local_rank if local_rank is not None else 0
        self.local_size = local_size if local_size is not None else 1
        self.cross_rank = cross_rank if cross_rank is not None else 0
        self.cross_size = cross_size if cross_size is not None else 1

        self._pub_port = pub_port + port_offset
        self._pull_port = pull_port + port_offset
        self._chief_ip = chief_ip

        self._is_chief = self.rank == 0
        self._is_local_chief = self.local_rank == 0

        if self.cross_size > 1:
            if chief_ip is None:
                raise AssertionError(
                    f"rank_info has cross_size ({self.cross_size}) but chief_ip was not "
                    "provided.  When cross_size > 1, the chief_ip parameter is required."
                )
            self._chief_ip = chief_ip
        else:
            # When cross_size == 1, always contact the chief as localhost.
            self._chief_ip = "127.0.0.1"

        self._closed = False

        self._init_ipc(force_tcp)

    def _init_ipc(self, force_tcp: bool) -> None:
        if self.size < 2:
            # No broadcasting necessary.
            return

        # Global broadcast server.
        if self._is_chief:
            logging.debug(f"Chief setting up server with ports {self._pub_port}/{self._pull_port}.")
            self._chief_zmq = ipc.ZMQBroadcastServer(
                num_connections=self.size - 1,
                pub_url=f"tcp://*:{self._pub_port}",
                pull_url=f"tcp://*:{self._pull_port}",
            )
            self._chief_zmq.safe_start(lambda: None)

        else:
            logging.debug(
                f"Non-Chief {self.rank} setting up comm to "
                f"{self._chief_ip} w/ ports "
                f"{self._pub_port}/{self._pull_port}."
            )
            self._worker_zmq = ipc.ZMQBroadcastClient(
                srv_pub_url=f"tcp://{self._chief_ip}:{self._pub_port}",
                srv_pull_url=f"tcp://{self._chief_ip}:{self._pull_port}",
            )
            self._worker_zmq.safe_start()

        if self.local_size < 2:
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
                num_connections=self.local_size - 1,
                pub_url=pub_url,
                pull_url=pull_url,
            )

            if pub_url is None:
                pub_url = f"tcp://localhost:{self._local_chief_zmq.get_pub_port()}"

            if pull_url is None:
                pull_url = f"tcp://localhost:{self._local_chief_zmq.get_pull_port()}"

            # Do a global allgather to initialize local clients on every node.
            local_chief = (self.cross_rank, pub_url, pull_url)
            _ = self._zmq_allgather(local_chief)
            self._local_chief_zmq.safe_start(lambda: None)

        else:
            # Start with the global allgather.
            all_local_chiefs = self._zmq_allgather(None)
            my_local_chief = [
                x for x in all_local_chiefs if x is not None and x[0] == self.cross_rank
            ]
            assert len(my_local_chief) == 1, (
                f"did not find exactly 1 local_chief for machine {self.cross_rank} "
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
        if self._closed or self.size < 2:
            return

        # Global broadcast server.
        if self._is_chief:
            self._chief_zmq.close()
        else:
            self._worker_zmq.close()

        if self.local_size < 2:
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
        return self.rank

    def get_local_rank(self) -> int:
        """
        Return the rank of the process on the agent. The local rank of a process
        is a unique ID within a given agent and trial; that is, no two processes
        in the same trial that are executing on the same agent will be assigned
        the same rank.
        """
        return self.local_rank

    def get_size(self) -> int:
        """
        Return the number of slots this trial is running on.
        """
        return self.size

    def get_num_agents(self) -> int:
        """
        Return the number of agents this trial is running on.
        """
        return self.cross_size

    def _zmq_gather(self, stuff: Any) -> Optional[List]:
        """
        Gather stuff to the chief.  The chief returns a list of all stuff, and workers return None.
        """
        if self.size < 2:
            return [stuff]
        logging.debug(f"Worker {self.get_rank()} beginning zmq gather.")
        if self._is_chief:
            worker_stuff_ranked, _ = self._chief_zmq.gather_with_polling(lambda: None)
            worker_stuff = [value for _, value in sorted(worker_stuff_ranked)]
            self._chief_zmq.broadcast(None)
            out = [stuff, *worker_stuff]  # type: Optional[List]
        else:
            self._worker_zmq.send((self.get_rank(), stuff))
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
        if self.local_size < 2:
            return [stuff]
        logging.debug(f"Worker {self.get_rank()} beginning zmq gather local.")
        if self._is_local_chief:
            worker_stuff_ranked, _ = self._local_chief_zmq.gather_with_polling(lambda: None)
            worker_stuff = [value for _, value in sorted(worker_stuff_ranked)]
            self._local_chief_zmq.broadcast(None)
            out = [stuff, *worker_stuff]  # type: Optional[List]
        else:
            self._local_worker_zmq.send((self.get_local_rank(), stuff))
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
        if self.size < 2:
            return [stuff]
        logging.debug(f"Worker {self.get_rank()} beginning zmq allgather.")
        if self._is_chief:
            worker_stuff_ranked, _ = self._chief_zmq.gather_with_polling(lambda: None)
            worker_stuff = [value for _, value in sorted(worker_stuff_ranked)]
            all_stuff = [stuff, *worker_stuff]
            self._chief_zmq.broadcast(all_stuff)
        else:
            self._worker_zmq.send((self.get_rank(), stuff))
            all_stuff = self._worker_zmq.recv()
        logging.debug(f"Worker {self.get_rank()} finished zmq allgather.")
        return all_stuff

    def _zmq_allgather_local(self, stuff: Any) -> List:
        """
        Gather stuff to the local chief and broadcast all of it back to the local workers.
        """
        if self.local_size < 2:
            return [stuff]
        logging.debug(f"Worker {self.get_rank()} beginning zmq local allgather.")
        if self._is_local_chief:
            worker_stuff_ranked, _ = self._local_chief_zmq.gather_with_polling(lambda: None)
            worker_stuff = [value for _, value in sorted(worker_stuff_ranked)]
            all_stuff = [stuff, *worker_stuff]
            self._local_chief_zmq.broadcast(all_stuff)
        else:
            self._local_worker_zmq.send((self.get_local_rank(), stuff))
            all_stuff = self._local_worker_zmq.recv()
        logging.debug(f"Worker {self.get_rank()} finished zmq local allgather.")
        return all_stuff

    def _zmq_broadcast(self, stuff: Any) -> Any:
        """
        Every worker gets the value sent by the chief.
        """
        if self.size < 2:
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
        if self.local_size < 2:
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


class DummyDistributed(DistributedContext):
    def __init__(self) -> None:
        super().__init__(
            rank=0,
            size=1,
            local_rank=0,
            local_size=1,
            cross_rank=0,
            cross_size=1,
        )
