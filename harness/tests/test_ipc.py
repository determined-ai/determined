import abc
import itertools
import multiprocessing
import sys
import textwrap
import threading
import time
import traceback
from typing import Any, Callable, List, Optional, cast

import pytest

import determined as det
from determined import ipc, layers, workload
from tests.experiment import utils
from tests.fixtures import fake_subprocess_receiver


class Subproc(multiprocessing.Process):
    """
    Subproc executes an abstract main(), returning the stacktrace as a string in join().
    """

    def __init__(self, *arg: Any, **kwarg: Any):
        self._error_queue = multiprocessing.Queue()  # type: Any
        super().__init__(*arg, **kwarg)

    def run(self) -> None:
        try:
            self.main()
        except Exception:
            self._error_queue.put(traceback.format_exc())

    def join_and_check(self, *args: Any, **kwargs: Any) -> Optional[str]:
        super().join(*args, **kwargs)
        if not self._error_queue.empty():
            return cast(str, self._error_queue.get())
        return None

    @abc.abstractmethod
    def main(self) -> None:
        pass


class SubprocGroup(list):
    """
    SubprocGroup provides a context manager to coordinate opening and closing of many Subprocs.
    """

    def join_all(self) -> None:
        # Every process should be joinable within one second.
        errors = [subproc.join_and_check() for subproc in self]

        # Terminate any processes which did not exit in time.
        num_unterminated = 0
        for subproc in self:
            if subproc.is_alive():
                subproc.terminate()
                subproc.join()
                num_unterminated += 1
        assert num_unterminated == 0

        # Make sure none of the processes raised an error.
        errors = [e for e in errors if e is not None]
        if len(errors):
            print("Traceback from child process:", file=sys.stderr)
            print(textwrap.indent(errors[0], "|"), file=sys.stderr)
            raise AssertionError("failure in child process")

    def __enter__(self) -> "SubprocGroup":
        for subproc in self:
            subproc.start()
        return self

    def __exit__(self, *_: Any) -> None:
        self.join_all()


class BroadcastClientSubproc(Subproc):
    def __init__(
        self, rank: int, size: int, pub_url: str, pull_url: str, exp_msgs: List[int]
    ) -> None:
        self._rank = rank
        self._size = size
        self._pub_url = pub_url
        self._pull_url = pull_url
        self._exp_msgs = exp_msgs
        super().__init__()

    def main(self) -> None:
        with ipc.ZMQBroadcastClient(self._pub_url, self._pull_url) as broadcast_client:
            # Start the server-client communication test.
            broadcast_client.send(ipc.ConnectedMessage(process_id=0))
            for exp in self._exp_msgs:
                msg = broadcast_client.recv()
                assert msg == exp
                broadcast_client.send(2 * msg)


def test_broadcast_server_client() -> None:
    num_subprocs = 3

    with ipc.ZMQBroadcastServer(num_connections=num_subprocs) as broadcast_server:

        pub_url = f"tcp://localhost:{broadcast_server.get_pub_port()}"
        pull_url = f"tcp://localhost:{broadcast_server.get_pull_port()}"
        msgs = list(range(10))

        with SubprocGroup(
            BroadcastClientSubproc(i, num_subprocs, pub_url, pull_url, msgs)
            for i in range(num_subprocs)
        ) as subprocs:

            def health_check() -> None:
                assert all(subproc.is_alive() for subproc in subprocs)
                for subproc in subprocs:
                    assert subproc.is_alive()

            gathered, _ = broadcast_server.gather_with_polling(health_check)
            assert all(isinstance(g, ipc.ConnectedMessage) for g in gathered)

            for msg in msgs:
                broadcast_server.broadcast(msg)
                gathered, _ = broadcast_server.gather_with_polling(health_check)
                assert all(g == 2 * msg for g in gathered)


def test_subprocess_launcher_receiver() -> None:
    hparams = {"global_batch_size": 1}
    exp_config = utils.make_default_exp_config(hparams, scheduling_unit=1, searcher_metric="loss")
    env = utils.make_default_env_context(hparams, exp_config)
    rendezvous_info = utils.make_default_rendezvous_info()
    hvd_config = utils.make_default_hvd_config()

    def make_workloads() -> workload.Stream:
        interceptor = workload.WorkloadResponseInterceptor()
        for i, wkld in enumerate(fake_subprocess_receiver.fake_workload_gen()):
            yield from interceptor.send(wkld)
            assert interceptor.metrics_result() == {"count": i}

    subproc = layers.SubprocessLauncher(
        env=env,
        workloads=make_workloads(),
        rendezvous_info=rendezvous_info,
        hvd_config=hvd_config,
        python_subprocess_entrypoint="tests.fixtures.fake_subprocess_receiver",
    )
    subproc.run()


def test_zmq_server_client() -> None:
    server = ipc.ZMQServer(num_connections=1, ports=None, port_range=(1000, 65535))
    assert len(server.get_ports()) == 1
    port = server.get_ports()[0]
    assert 1000 <= port <= 65535

    client = ipc.ZMQClient(ip_address="localhost", port=port)

    client_object = {"DeterminedAI": "Great", "det": "Fantastic", 12345: -100}
    client.send(client_object)
    server_object = server.receive_blocking(send_rank=0)
    assert server_object == client_object

    server_object["DeterminedAI"] = "VeryGreat"
    server.send(server_object)
    client_object = client.receive()
    assert server_object == client_object


@pytest.mark.parametrize("cross_size", [1, 4])  # type: ignore
@pytest.mark.parametrize("local_size", [1, 4])  # type: ignore
@pytest.mark.parametrize("force_tcp", [False, True])  # type: ignore
def test_distributed_context(cross_size: int, local_size: int, force_tcp: bool) -> None:
    size = cross_size * local_size
    # Generate one rendezvous_info per node.
    rendezvous_info = [
        det.RendezvousInfo(
            addrs=["localhost:12345"] * cross_size,
            rank=i,
        )
        for i in range(cross_size)
    ]

    def do_parallel(fn: Callable) -> List:
        """
        Run the same function on one-thread-per-rank, assert there were no exceptions, and return
        the results from each rank.
        """
        results = [None] * size  # type: List
        errors = [None] * size  # type: List
        threads = []

        for cross_rank, local_rank in itertools.product(range(cross_size), range(local_size)):
            rank = cross_rank * local_size + local_rank

            def _fn(rank: int, cross_rank: int, local_rank: int) -> None:
                try:
                    results[rank] = fn(rank, cross_rank, local_rank)
                except Exception:
                    errors[rank] = traceback.format_exc()
                    raise

            threads.append(threading.Thread(target=_fn, args=(rank, cross_rank, local_rank)))

        for thread in threads:
            thread.start()

        for thread in threads:
            thread.join()

        assert errors == [None] * size, "not all threads exited without error"

        return results

    # Create all of the DistributedContexts.
    def make_distributed_context(rank: int, cross_rank: int, local_rank: int) -> Any:
        rank_info = det.RankInfo(
            rank=cross_rank * local_size + local_rank,
            size=cross_size * local_size,
            local_rank=local_rank,
            local_size=local_size,
            cross_rank=cross_rank,
            cross_size=cross_size,
        )
        return det.DistributedContext(
            rank_info=rank_info,
            rendezvous_info=rendezvous_info[cross_rank],
            unique_port_offset=0,
            force_tcp=force_tcp,
        )

    contexts = do_parallel(make_distributed_context)

    # Perform a broadcast.
    results = do_parallel(lambda rank, _, __: contexts[rank]._zmq_broadcast(rank))
    assert results == [0] * size, "not all threads ran broadcast correctly"

    # Perform a local broadcast.
    results = do_parallel(lambda rank, _, __: contexts[rank]._zmq_broadcast_local(rank))
    expect = [rank - (rank % local_size) for rank in range(size)]  # type: Any

    assert results == expect, "not all threads ran broadcast_local correctly"

    # Perform a gather.
    results = do_parallel(lambda rank, _, __: set(contexts[rank]._zmq_gather(rank) or []))
    chief = set(range(size))
    expect = [set(range(size)) if rank == 0 else set() for rank in range(size)]
    assert results == [chief] + [set()] * (size - 1), "not all threads ran gather correctly"

    # Perform a local gather.
    results = do_parallel(lambda rank, _, __: set(contexts[rank]._zmq_gather_local(rank) or []))
    expect = [
        set(range(rank, rank + local_size)) if rank % local_size == 0 else set()
        for rank in range(size)
    ]
    assert results == expect, "not all threads ran gather correctly"

    # Perform an allgather.
    results = do_parallel(lambda rank, _, __: set(contexts[rank]._zmq_allgather(rank)))
    expect = set(range(size))
    assert results == [expect] * size, "not all threads ran allgather correctly"

    # Perform a local allgather.
    results = do_parallel(lambda rank, _, __: set(contexts[rank]._zmq_allgather_local(rank)))
    expect = [
        set(range(cross_rank * local_size, (cross_rank + 1) * local_size))
        for cross_rank, _ in itertools.product(range(cross_size), range(local_size))
    ]
    assert results == expect, "not all threads ran allgather_local correctly"

    # Close all contexts.
    for context in contexts:
        context.close()


class TestPIDServer:
    def test_normal_execution(self) -> None:
        with ipc.PIDServer(addr=0, num_clients=2) as pid_server:
            assert pid_server.listener
            _, port = pid_server.listener.getsockname()

            def worker_proc() -> None:
                with ipc.PIDClient(port) as pid_client:
                    for _ in range(5):
                        pid_client.keep_alive()
                        time.sleep(0.1)

            procs = [
                multiprocessing.Process(target=worker_proc),
                multiprocessing.Process(target=worker_proc),
            ]

            for p in procs:
                p.start()

            pid_server.run()

            for p in procs:
                p.join()

            assert len(pid_server.graceful_shutdowns) == 2

    def test_worker_crashes(self) -> None:
        with ipc.PIDServer(addr=0, num_clients=2) as pid_server:
            assert pid_server.listener
            _, port = pid_server.listener.getsockname()

            # Enforce that the crashed worker causes the exit before the other worker exits.
            deadline = time.time() + 20

            def worker_proc() -> None:
                with ipc.PIDClient(port):
                    # Wait for the crashing process to cause us to die.
                    time.sleep(30)

            def crashing_worker_proc() -> None:
                with ipc.PIDClient(port):
                    time.sleep(0.5)
                    raise ValueError("Crashing...")

            procs = [
                multiprocessing.Process(target=worker_proc),
                multiprocessing.Process(target=crashing_worker_proc),
            ]

            for p in procs:
                p.start()

            with pytest.raises(det.errors.WorkerError):
                pid_server.run()

            assert time.time() < deadline, "crashing worker did not trigger exit"

            for p in procs:
                p.terminate()
                p.join()

            assert len(pid_server.graceful_shutdowns) == 0

    def test_health_check_pre_connect(self) -> None:
        with ipc.PIDServer(addr=0, num_clients=2) as pid_server:
            assert pid_server.listener
            _, port = pid_server.listener.getsockname()

            fail_time = time.time() + 0.2

            def worker_proc() -> None:
                with ipc.PIDClient(port):
                    time.sleep(10)

            def health_check() -> None:
                assert time.time() < fail_time

            # Only one worker to guarantee a failed healthcheck before all workers have connected.
            procs = [
                multiprocessing.Process(target=worker_proc),
            ]

            for p in procs:
                p.start()

            with pytest.raises(AssertionError):
                pid_server.run(health_check, poll_period=0.05)

            for p in procs:
                p.join()

            assert len(pid_server.graceful_shutdowns) == 0

    def test_health_check_post_connect(self) -> None:
        with ipc.PIDServer(addr=0, num_clients=2) as pid_server:
            assert pid_server.listener
            _, port = pid_server.listener.getsockname()

            fail_time = time.time() + 0.2

            def worker_proc() -> None:
                with ipc.PIDClient(port):
                    time.sleep(10)

            def health_check() -> None:
                assert time.time() < fail_time

            procs = [
                multiprocessing.Process(target=worker_proc),
                multiprocessing.Process(target=worker_proc),
            ]

            for p in procs:
                p.start()

            with pytest.raises(AssertionError):
                pid_server.run(health_check, poll_period=0.05)

            for p in procs:
                p.join()

            assert len(pid_server.graceful_shutdowns) == 0
