import abc
import itertools
import multiprocessing
import sys
import textwrap
import time
import traceback
from typing import Any, List, Optional, cast

import pytest

import determined as det
from determined import core, ipc
from tests import parallel


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
            broadcast_client.safe_start()
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
        ):
            broadcast_server.safe_start()
            for msg in msgs:
                broadcast_server.broadcast(msg)
                gathered = broadcast_server.gather()
                assert all(g == 2 * msg for g in gathered)


@pytest.mark.parametrize("cross_size", [1, 4])
@pytest.mark.parametrize("local_size", [1, 4])
@pytest.mark.parametrize("force_tcp", [False, True])
def test_distributed_context(cross_size: int, local_size: int, force_tcp: bool) -> None:
    size = cross_size * local_size

    # Make sure `make test` doesn't hang on macbook's default values.  Avoid skipping on linux
    # because it's not a common default, and to avoid false positives in CI.
    if sys.platform == "darwin" and size == 16:
        import resource

        if resource.getrlimit(resource.RLIMIT_NOFILE)[0] < 1024:
            pytest.skip(
                "increase the open fd limit with `ulimit -n 1024` or greater to run this test"
            )

    with parallel.Execution(size, local_size=local_size, make_distributed_context=False) as pex:

        @pex.run
        def contexts() -> core.DistributedContext:
            return core.DistributedContext(
                rank=pex.rank,
                size=pex.size,
                local_rank=pex.local_rank,
                local_size=pex.local_size,
                cross_rank=pex.cross_rank,
                cross_size=pex.cross_size,
                chief_ip="localhost",
                force_tcp=force_tcp,
            )

        # Perform a broadcast.
        results = pex.run(lambda: contexts[pex.rank].broadcast(pex.rank))
        assert results == [0] * size, "not all threads ran broadcast correctly"

        # Perform a local broadcast.
        results = pex.run(lambda: contexts[pex.rank].broadcast_local(pex.rank))
        expect = [rank - (rank % local_size) for rank in range(size)]  # type: Any

        assert results == expect, "not all threads ran broadcast_local correctly"

        # Perform a gather.
        results = pex.run(lambda: set(contexts[pex.rank].gather(pex.rank) or []))
        chief = set(range(size))
        expect = [set(range(size)) if rank == 0 else set() for rank in range(size)]
        assert results == [chief] + [set()] * (size - 1), "not all threads ran gather correctly"

        # Perform a local gather.
        results = pex.run(lambda: set(contexts[pex.rank].gather_local(pex.rank) or []))
        expect = [
            set(range(rank, rank + local_size)) if rank % local_size == 0 else set()
            for rank in range(size)
        ]
        assert results == expect, "not all threads ran gather correctly"

        # Perform an allgather.
        results = pex.run(lambda: set(contexts[pex.rank].allgather(pex.rank)))
        expect = set(range(size))
        assert results == [expect] * size, "not all threads ran allgather correctly"

        # Perform a local allgather.
        results = pex.run(lambda: set(contexts[pex.rank].allgather_local(pex.rank)))
        expect = [
            set(range(cross_rank * local_size, (cross_rank + 1) * local_size))
            for cross_rank, _ in itertools.product(range(cross_size), range(local_size))
        ]
        assert results == expect, "not all threads ran allgather_local correctly"

        # Close all contexts.
        for context in contexts:
            context.close()


class TestPIDServer:
    @staticmethod
    def _worker_proc(
        addr: int,
        keep_alive: bool = False,
        sleep_time: float = 10,
        repeat: int = 1,
        crash: bool = False,
    ) -> None:
        with ipc.PIDClient(addr) as pid_client:
            for _ in range(repeat):
                if keep_alive:
                    pid_client.keep_alive()
                time.sleep(sleep_time)
            if crash:
                raise ValueError("Crashing...")

    def test_normal_execution(self) -> None:
        with ipc.PIDServer(addr=0, num_clients=2) as pid_server:
            assert pid_server.listener
            _, port = pid_server.listener.getsockname()

            procs = [
                multiprocessing.Process(
                    target=TestPIDServer._worker_proc, args=(port, True, 0.1, 5)
                ),
                multiprocessing.Process(
                    target=TestPIDServer._worker_proc, args=(port, True, 0.1, 5)
                ),
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

            procs = [
                multiprocessing.Process(target=TestPIDServer._worker_proc, args=(port, False, 30)),
                multiprocessing.Process(
                    target=TestPIDServer._worker_proc, args=(port, False, 0.5, 1, True)
                ),
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

    def test_return_code_on_worker_error(self) -> None:
        with ipc.PIDServer(addr=0, num_clients=2) as pid_server:
            assert pid_server.listener
            _, port = pid_server.listener.getsockname()

            # Enforce that the crashed worker causes the exit before the other worker exits.
            deadline = time.time() + 20

            # Enforce that run_subprocess exits nonzero on a worker failure, even if the main
            # subprocess exits zero.
            procs = [
                multiprocessing.Process(target=TestPIDServer._worker_proc, args=(port, False, 30)),
                multiprocessing.Process(
                    target=TestPIDServer._worker_proc, args=(port, False, 0.5, 1, True)
                ),
            ]

            for p in procs:
                p.start()

            error_code = pid_server.run_subprocess(["sleep", "2"])

            assert error_code == 79

            assert time.time() < deadline, "crashing worker did not trigger exit"

            for p in procs:
                p.terminate()
                p.join()

    def test_health_check_pre_connect(self) -> None:
        with ipc.PIDServer(addr=0, num_clients=2) as pid_server:
            assert pid_server.listener
            _, port = pid_server.listener.getsockname()

            fail_time = time.time() + 0.2

            def health_check() -> None:
                assert time.time() < fail_time

            # Only one worker to guarantee a failed healthcheck before all workers have connected.
            procs = [
                multiprocessing.Process(target=TestPIDServer._worker_proc, args=(port, False)),
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

            def health_check() -> None:
                assert time.time() < fail_time

            procs = [
                multiprocessing.Process(target=TestPIDServer._worker_proc, args=(port, False)),
                multiprocessing.Process(target=TestPIDServer._worker_proc, args=(port, False)),
            ]

            for p in procs:
                p.start()

            with pytest.raises(AssertionError):
                pid_server.run(health_check, poll_period=0.05)

            for p in procs:
                p.join()

            assert len(pid_server.graceful_shutdowns) == 0

    def test_single_worker_failure_is_caught(self) -> None:
        # This is a regression test; there used to be a codepath where we would stop checking pid's
        # after the last pidclient disconnected, even if it disconnected with a failure.
        with ipc.PIDServer(addr=0, num_clients=1) as pid_server:
            assert pid_server.listener
            _, port = pid_server.listener.getsockname()

            p = multiprocessing.Process(
                target=TestPIDServer._worker_proc, args=(port, False, 0.5, 1, True)
            )

            p.start()

            with pytest.raises(det.errors.WorkerError):
                pid_server.run()

            p.terminate()
            p.join()
