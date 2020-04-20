import abc
import multiprocessing
import sys
import textwrap
import traceback
from typing import Any, List, Optional, cast

from determined import ipc, layers, workload
from tests.unit.fixtures import fake_subprocess_receiver
from tests.unit.frameworks import utils


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
            broadcast_client.send(ipc.ReadyMessage())
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

            gathered = broadcast_server.gather_with_polling(health_check)
            assert all(isinstance(g, ipc.ReadyMessage) for g in gathered)

            for msg in msgs:
                broadcast_server.broadcast(msg)
                gathered = broadcast_server.gather_with_polling(health_check)
                assert all(g == 2 * msg for g in gathered)


def test_subprocess_launcher_receiver() -> None:
    env = utils.make_default_env_context(hparams={"global_batch_size": 1})
    rendezvous_info = utils.make_default_rendezvous_info()
    hvd_config = utils.make_default_hvd_config()

    def make_workloads() -> workload.Stream:
        interceptor = workload.WorkloadResponseInterceptor()
        for i, wkld in enumerate(fake_subprocess_receiver.fake_workload_gen()):
            yield from interceptor.send(wkld, [])
            assert interceptor.metrics_result() == {"count": i}

    subproc = layers.SubprocessLauncher(
        env=env,
        workloads=make_workloads(),
        load_path=None,
        rendezvous_info=rendezvous_info,
        hvd_config=hvd_config,
        python_subprocess_entrypoint="tests.unit.fixtures.fake_subprocess_receiver",
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
