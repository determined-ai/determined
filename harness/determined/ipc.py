import logging
import os
import selectors
import signal
import socket
import subprocess
import time
from typing import Any, Callable, Dict, List, Optional, Tuple, Union, cast

import psutil
import zmq
from zmq.error import ZMQBindError, ZMQError

import determined as det
from determined.common import check


class _OneSidedBarrier:
    """
    _OneSidedBarrier is a message from participants (usually workers) to a single process (usually
    the chief) indicating to the chief that the workers are ready for the next phase of
    computation.
    """

    def __init__(self, message: Any) -> None:
        self.message = message


class _HelloMessage:
    pass


class _FinalHelloMessage:
    pass


class ConnectedMessage:
    """
    ConnectedMessage is sent by a SubprocessReceiver to the SubprocessLauncher when it is starts
    and contains the SubprocessReceiver's process id so that the SubprocessLauncher can
    perform health check on it.
    """

    def __init__(self, process_id: int) -> None:
        self.process_id = process_id


class _SerialMessage:
    """
    _SerialMessage wraps a payload in a monotonically-increasing serial number, which makes it easy
    to confirm that our broadcasting does not get out-of-sync.
    """

    def __init__(self, serial: int, payload: Any) -> None:
        self.serial = serial
        self.payload = payload


class _ExceptionMessage:
    """
    _ExceptionMessage is sent by a training subprocess to indicate that an exception has occurred.
    """

    pass


class ZMQBroadcastServer:
    """
    Similar to ZMQServer except with broadcast/gather semantics on exactly two ports.

    Using a constant number of ports allows the SubprocessReceiver to be configured without knowing
    any rank information (i.e., before user code is read and horovod can be initialized).

    A ConnectedMessage must be observed from each connection before it is safe to broadcast. This
    can be accomplished by calling gather_with_polling() and checking that all gathered messages
    are ConnectedMessages.

    ZMQBroadcastServer uses ZMQ PUB-SUB pattern to transmit messages to worker processes, and uses
    the PUSH-PULL pattern to collect responses from workers. The reason for this asymmetry is that
    PUSH-PULL connections block during send() if the remote end is dead. Therefore, PUSH-PULL
    cannot be used to transmitting from server to worker, because if all the workers die, the
    server would hang.

    Additionally, the server can't receive messages from workers via the PUB-SUB pattern, because
    the startup semantics of PUB-SUB connections in ZMQ are slightly odd; the SUB socket must
    connect to the PUB socket.  Normally this happens when you do sub_socket.connect(), but if the
    server creates a SUB socket and does sub_socket.bind(), then when the client creates a PUB
    socket and calls pub_socket.connect(), ZMQ still has to create a connection from the SUB to the
    PUB (since sub_socket used bind() instead of connect()) and the server's SUB socket will
    usually miss the first message sent by the client's PUB socket.

    See ZMQ documentation for a related discussion on PUB-SUB sockets:
    http://zguide.zeromq.org/page:all#Getting-the-Message-Out (look for "one more important thing")
    http://zguide.zeromq.org/page:all#Node-Coordination
    (link broke, use http://web.archive.org/web/20191011190012/http://zguide.zeromq.org/page:all)
    """

    def __init__(
        self, num_connections: int, pub_url: Optional[str] = None, pull_url: Optional[str] = None
    ) -> None:
        self._num_connections = num_connections

        context = zmq.Context()  # type: ignore

        self._pub_socket = context.socket(zmq.PUB)
        self._pull_socket = context.socket(zmq.PULL)

        self._pub_port = None  # type: Optional[int]
        self._pull_port = None  # type: Optional[int]

        if pub_url is None:
            self._pub_port = self._pub_socket.bind_to_random_port("tcp://*")
        else:
            self._pub_socket.bind(pub_url)

        if pull_url is None:
            self._pull_port = self._pull_socket.bind_to_random_port("tcp://*")
        else:
            self._pull_socket.bind(pull_url)

        self._send_serial = 0
        self._recv_serial = 0

    def safe_start(self, health_check: Callable[[], None]) -> None:
        """
        Broadcast Hello messages over and over until all clients response with a Hello message.

        The reason for this is that the only way to be 100% confident that a subscriber has
        connected is for it to actually receive a message over the pub/sub connection.

        After each client sees its first Hello, it will send a single Hello message to the
        server.

        After all connections have been made, the server will broadcast a FinalHello.
        """

        connections_made = 0
        while connections_made < self._num_connections:
            # Send a Hello.
            self._pub_socket.send_pyobj(_HelloMessage())

            # Check for an incoming connection.
            if self._pull_socket.poll(50) == 0:
                health_check()
                continue

            obj = self._pull_socket.recv_pyobj()
            check.is_instance(obj, _HelloMessage, "got non-_HelloMessage in server safe_start")
            connections_made += 1

        self._pub_socket.send_pyobj(_FinalHelloMessage())

    def __enter__(self) -> "ZMQBroadcastServer":
        return self

    def __exit__(self, *_: Any) -> None:
        self.close()

    def close(self) -> None:
        self._pub_socket.close()
        self._pull_socket.close()

    def get_pub_port(self) -> int:
        if self._pub_port is None:
            raise ValueError("get_pub_port() is only safe when pub_url was None")
        return self._pub_port

    def get_pull_port(self) -> int:
        if self._pull_port is None:
            raise ValueError("get_pull_port() is only safe when pull_url was None")
        return self._pull_port

    def broadcast(self, obj: Any) -> None:
        """
        Broadcast a message object to each connection.
        """

        self._pub_socket.send_pyobj(_SerialMessage(self._send_serial, obj))
        self._send_serial += 1

    def gather_with_polling(self, health_check: Callable[[], None]) -> Tuple[List[Any], bool]:
        """
        Gather a response message from each connection, with a health_check callback that can raise
        an error if something goes wrong. Returns list of messages and whether any of the senders
        indicate an exception.
        """
        messages = []  # type: List[Any]
        while len(messages) < self._num_connections:
            if self._pull_socket.poll(1000) == 0:
                # Call the polling function (probably check if a subprocess is alive).
                health_check()
                continue

            message, message_type = self._recv_one()
            messages.append(message)

            if message_type is _ExceptionMessage:
                return messages, True

        self._recv_serial += 1

        return messages, False

    def _recv_one(self) -> Tuple[Any, type]:
        """
        Receive one _SerialMessage from the socket and confirm that it is in-order.
        """

        obj = self._pull_socket.recv_pyobj()

        if isinstance(obj, _ExceptionMessage):
            return None, _ExceptionMessage

        if isinstance(obj, _SerialMessage):
            check.eq(obj.serial, self._recv_serial, "Out-of-order client message detected")
            return obj.payload, _SerialMessage

        raise AssertionError(f"Unexpected message type encountered: {type(obj)}")


class ZMQBroadcastClient:
    def __init__(self, srv_pub_url: str, srv_pull_url: str) -> None:
        context = zmq.Context()  # type: ignore

        self._sub_socket = context.socket(zmq.SUB)
        # Subscriber always listens to ALL messages.
        self._sub_socket.subscribe(b"")
        self._sub_socket.connect(srv_pub_url)

        self._push_socket = context.socket(zmq.PUSH)
        self._push_socket.connect(srv_pull_url)

        self._send_serial = 0
        self._recv_serial = 0

    def safe_start(self) -> None:
        """
        See ZMQBroadcastServer.safe_start().
        """

        # Get the first HelloMessage to guarantee our SUB socket is connected.
        obj = self._sub_socket.recv_pyobj()
        check.is_instance(obj, _HelloMessage, "got non-HelloMessage in client.safe_start()")

        # Send our own _HelloMessage.
        self._push_socket.send_pyobj(_HelloMessage())

        while True:
            # Discard all further Hellos until the FinalHello.
            obj = self._sub_socket.recv_pyobj()
            if isinstance(obj, _FinalHelloMessage):
                break
            check.is_instance(obj, _HelloMessage, "got non-HelloMessage in client.safe_start()")

    def __enter__(self) -> "ZMQBroadcastClient":
        return self

    def __exit__(self, *_: Any) -> None:
        self.close()

    def close(self) -> None:
        self._sub_socket.close()
        self._push_socket.close()

    def send(self, obj: Any) -> None:
        message = _SerialMessage(self._send_serial, obj)
        self._send_serial += 1
        self._push_socket.send_pyobj(message)

    def send_exception_message(self) -> None:
        message = _ExceptionMessage()
        self._push_socket.send_pyobj(message)

    def recv(self) -> Any:

        obj = self._sub_socket.recv_pyobj()

        if isinstance(obj, _SerialMessage):
            check.eq(obj.serial, self._recv_serial, "Out-of-order server message detected")
            self._recv_serial += 1
            return obj.payload
        raise AssertionError(f"Unexpected message type encountered: {type(obj)}")


class ZMQServer:
    """
    ZMQServer connects the trial controller with training subprocesses.
    It also synchronizes the chief trial runner with all non-chief trial
    runners when using Horovod.

    For communicating with training subprocess, we initialize a separate
    socket (which binds to a unique port) for each connection.
    Clients connecting to the ZMQ server (workers or non-chief trial controllers)
    need to send the initial message, and each socket needs to have a strict
    send-receive message ordering (a requirement for ZMQ REQ and REP sockets).

    ZMQServer takes as input either a list of specific ports, or a range of ports.
    If a range of ports  is specified,  ZMQ will randomly select an available port
    within the range.
    """

    def __init__(
        self,
        num_connections: Optional[int] = None,
        ports: Optional[List[int]] = None,
        port_range: Optional[Tuple[int, int]] = None,
    ) -> None:
        self.context = zmq.Context()  # type: ignore
        self.sockets = []  # type: List[zmq.Socket]
        self.ports = []  # type: List[int]

        if ports:
            check.is_none(port_range)
            self._bind_to_specified_ports(ports=ports)
            check.eq(len(self.ports), len(ports))
        else:
            check.is_not_none(num_connections)
            check.is_not_none(port_range)
            num_connections = cast(int, num_connections)
            port_range = cast(Tuple[int, int], port_range)
            self._bind_to_random_ports(port_range=port_range, num_connections=num_connections)
            check.eq(len(self.ports), num_connections)

    def __enter__(self) -> "ZMQServer":
        return self

    def __exit__(self, *_: Any) -> None:
        self.close()

    def _bind_to_specified_ports(self, ports: List[int]) -> None:
        for port in ports:
            socket = self.context.socket(zmq.REP)
            try:
                socket.bind(f"tcp://*:{port}")
            except ZMQError as e:
                raise det.errors.InternalException(f"Failed to bind to port {port}.") from e
            self.sockets.append(socket)
            self.ports.append(port)

    def _bind_to_random_ports(self, port_range: Tuple[int, int], num_connections: int) -> None:
        check.lt(num_connections, port_range[1] - port_range[0])
        for _ in range(num_connections):
            socket = self.context.socket(zmq.REP)
            try:
                selected_port = socket.bind_to_random_port(
                    addr="tcp://*", min_port=port_range[0], max_port=port_range[1]
                )
                self.ports.append(selected_port)
            except ZMQBindError as e:
                raise det.errors.InternalException(
                    f"Failed to bind to port range {port_range}."
                ) from e
            self.sockets.append(socket)

    def get_ports(self) -> List[int]:
        return self.ports

    def send(self, py_obj: Any) -> None:
        for sock in self.sockets:
            sock.send_pyobj(py_obj)  # type: ignore

    def receive_blocking(self, send_rank: int) -> Any:
        check.lt(send_rank, len(self.sockets))
        message = self.sockets[send_rank].recv_pyobj()  # type: ignore
        return message

    def receive_non_blocking(
        self, send_rank: int, deadline: Optional[float] = None
    ) -> Tuple[bool, Any]:
        check.lt(send_rank, len(self.sockets))
        timeout = 1000 if not deadline else int(deadline - time.time()) * 1000
        timeout = max(timeout, 100)

        if self.sockets[send_rank].poll(timeout) == 0:  # type: ignore
            return False, None
        message = self.sockets[send_rank].recv_pyobj()  # type: ignore
        return True, message

    def barrier(
        self, num_connections: int, message: Any = None, timeout: Optional[int] = None
    ) -> List[Any]:
        """
        This is a one-sided barrier, where the chief blocks until
        all non-chief trial containers have sent a message.
        """
        check.eq(len(self.sockets), 1)
        messages = []  # type: List[Any]
        start_time = time.time()

        for _ in range(num_connections):
            if timeout:
                message_received, barrier_message = self.receive_non_blocking(
                    send_rank=0, deadline=start_time + timeout
                )

                if not message_received:
                    return messages

            else:
                barrier_message = self.receive_blocking(0)

            check.is_instance(barrier_message, _OneSidedBarrier)
            messages.append(barrier_message.message)
            self.sockets[0].send_pyobj(_OneSidedBarrier(message=message))  # type: ignore

        return messages

    def close(self) -> None:
        for sock in self.sockets:
            sock.close()  # type: ignore


class ZMQClient:
    """
    ZMQClient connects training subprocesses with trial-controller.
    It also signals the chief trial-controller, when the non-chief
    trial controller has successfully started sshd.
    """

    def __init__(self, ip_address: str, port: int) -> None:
        self.context = zmq.Context()  # type: ignore
        self.socket = self.context.socket(zmq.REQ)
        self.socket.connect(f"tcp://{ip_address}:{port}")

    def __enter__(self) -> "ZMQClient":
        return self

    def __exit__(self, *_: Any) -> None:
        self.close()

    def send(self, py_obj: Any) -> None:
        self.socket.send_pyobj(py_obj)

    def receive(self) -> Any:
        return self.socket.recv_pyobj()

    def barrier(self, message: Any = None) -> Any:
        """
        This is a one-sided barrier, where the chief blocks until
        all non-chief trial containers have sent a message.
        """
        self.socket.send_pyobj(_OneSidedBarrier(message=message))
        barrier_message = self.socket.recv_pyobj()
        check.is_instance(barrier_message, _OneSidedBarrier)
        return barrier_message.message

    def close(self) -> None:
        self.socket.close()


def read_pid_server_addr(addr: str) -> Union[str, int, Tuple[str, int]]:
    """
    Read a string for specifying either a unix socket, a port number, or a host:port string.

    Used by both the pid_server and pid_client helper scripts.
    """
    if "/" in addr:
        # Unix socket.
        return addr
    if ":" in addr:
        # Host:port string.
        parts = addr.split(":")
        host = ":".join(parts[:-1])
        port = int(parts[-1])
        return host, port
    try:
        return int(addr)
    except ValueError:
        pass
    raise ValueError(
        "'{addr}' is not a valid address spec; it should be a path to a unix socket (with at least "
        "one '/'), a host:port string, or a port number"
    )


class PIDServer:
    """
    PIDServer tracks PIDs reported by a set of pid_clients which connect to it.

    PIDServer.run() will return when all pid_clients have reported a graceful shutdown and have
    exited, or it will raise an exception if any pids disappear without reporting a graceful
    shutdown.

    PIDServer lets an sshd-based launch layer keep track of its worker processes, even when the
    worker processes aren't proper child processes.
    """

    def __init__(self, addr: Union[str, int, Tuple[str, int]], num_clients: int) -> None:
        self.addr = addr
        self.num_clients = num_clients

        self.started = False
        self.sel = None  # type: Optional[selectors.BaseSelector]
        self.listener = None  # type: Optional[socket.socket]

        self.pids = []  # type: List[int]
        self.graceful_shutdowns = []  # type: List[int]
        # maps a connection to its pid
        self.conns = {}  # type: Dict[socket.socket, int]

        self.done_accepting = False

    def start(self) -> "PIDServer":
        if self.started:
            return self
        self.started = True
        try:
            self.sel = selectors.DefaultSelector()
            if isinstance(self.addr, str):
                # Unix socket.
                self.listener = socket.socket(family=socket.AF_UNIX)
                if os.path.exists(self.addr):
                    os.remove(self.addr)
                self.listener.bind(self.addr)
            elif isinstance(self.addr, int):
                # A TCP Port.
                self.listener = socket.socket()
                self.listener.bind(("", self.addr))
            else:
                # A address and a port.
                self.listener = socket.socket()
                self.listener.bind(self.addr)
            self.listener.listen(self.num_clients)
            self.listener.setblocking(False)
            self.sel.register(self.listener, selectors.EVENT_READ)
            return self
        except Exception:
            self.close()
            raise

    def close(self) -> None:
        self.started = False
        if self.listener:
            self.listener.close()
            self.listener = None
        if self.sel:
            self.sel.close()
            self.sel = None

    def __enter__(self) -> "PIDServer":
        return self.start()

    def __exit__(self, *_: Any) -> None:
        self.close()

    def handle_listener(self, mask: int) -> None:
        """
        Handle an event on a listener socket (aka, accept a connection).
        """
        assert self.sel
        assert self.listener
        if mask & selectors.EVENT_READ:
            conn, _ = self.listener.accept()
            # We never write anything.
            conn.shutdown(socket.SHUT_WR)
            # First, receive the initial PID for this conn.  Should be nearly instant.
            buf = b""
            while b"\n" not in buf:
                data = conn.recv(4096)
                if not data:
                    raise ValueError("pid_client did not deliver a PID!")
                buf += data
            pid_buf, data_buf = buf.split(b"\n", 1)
            pid = int(pid_buf)
            self.pids.append(pid)
            self.conns[conn] = pid
            # Now listen for this connection to gracefully shut down (eventually)
            conn.setblocking(False)
            self.sel.register(conn, selectors.EVENT_READ)
            if len(self.pids) == self.num_clients:
                # That the last connection, close the listener.
                self.sel.unregister(self.listener)
                self.listener.close()
                self.listener = None
            if data_buf:
                # We received a message in the same packet as the PID, simulate an EVENT_READ.
                self.handle_conn(conn, mask=0, data=data_buf)
        else:
            raise ValueError("listener failed")

    def handle_conn(self, conn: socket.socket, mask: int, data: Optional[bytes] = None) -> None:
        """
        Handle an event on a connection socket.

        You can simulate an EVENT_READ on in-memory data by setting mask==0 and data!=None.
        """
        assert self.sel
        pid = self.conns[conn]
        if mask & selectors.EVENT_READ:
            data = conn.recv(4096)
        # Messages are all one-byte codes for easy parsing.
        # The protocol is "any number of keepalive "k"s followed by a quit "q", so we can
        # safely ignore everything except the final byte of the message.
        if data:
            if data[-1:] == b"k":
                # keepalive message; leave the connection alone.
                return
            elif data[-1:] == b"q":
                # Graceful shutdown code.
                self.graceful_shutdowns.append(pid)
            else:
                raise ValueError("invalid message from pid_client:", data)

        # Error, EOF, or anything else.

        if self.listener is not None:
            raise det.errors.WorkerError("worker died before all workers connected")

        self.sel.unregister(conn)
        conn.close()
        del self.conns[conn]

    def check_pids(self) -> None:
        """
        Any PIDs which exited without a graceful exit message indicates a crashed worker.
        """
        for pid in self.pids:
            if pid not in self.graceful_shutdowns:
                pid_ok = False
                try:
                    if psutil.Process(pid).status() not in (
                        psutil.STATUS_DEAD,
                        psutil.STATUS_STOPPED,
                        psutil.STATUS_ZOMBIE,
                    ):
                        pid_ok = True
                except psutil.NoSuchProcess:
                    pass
                if not pid_ok:
                    raise det.errors.WorkerError("Detected that worker process died.")

    def run(self, health_check: Optional[Callable] = None, poll_period: float = 1) -> None:
        assert self.sel, "must start first"
        # Continue until we aren't waiting for anything else to shut down.
        while self.listener or self.conns:
            # Get some read events.
            for key, mask in self.sel.select(timeout=poll_period):
                if key.fileobj == self.listener:
                    self.handle_listener(mask)
                elif key.fileobj in self.conns:
                    conn = key.fileobj
                    assert isinstance(conn, socket.socket)
                    self.handle_conn(conn, mask)
                else:
                    raise AssertionError(f"unexpected key from select(): {key}")

            self.check_pids()

            # If all workers exited gracefully, shut down nicely.
            if len(self.graceful_shutdowns) == self.num_clients:
                return

            # Otherwise, run the externally-provided health check.
            if health_check is not None:
                health_check()

    def run_subprocess(
        self,
        cmd: List[str],
        on_fail: Optional[signal.Signals] = None,
        on_exit: Optional[signal.Signals] = None,
        grace_period: int = 3,
    ) -> int:
        p = subprocess.Popen(cmd)

        class HealthCheckFail(Exception):
            def __init__(self, exit_code: int):
                super().__init__()
                self.exit_code = exit_code

        def health_check() -> None:
            ret = p.poll()
            if ret is not None:
                raise HealthCheckFail(ret)

        try:
            self.run(health_check)
        except HealthCheckFail as e:
            return e.exit_code
        except det.errors.WorkerError:
            # Worker failed.
            if on_fail is not None:
                # Let things finish logging, exiting on their own, etc.
                time.sleep(grace_period)
                p.send_signal(on_fail)
                if on_fail != signal.SIGKILL:
                    try:
                        return p.wait(timeout=10)
                    except subprocess.TimeoutExpired:
                        logging.error(f"killing worker which didn't exit after {on_fail.name}")
                        p.send_signal(signal.SIGKILL)
            return p.wait()

        # All workers exited normally.
        if on_exit is not None:
            time.sleep(grace_period)
            p.send_signal(on_exit)
            if on_exit != signal.SIGKILL:
                try:
                    return p.wait(timeout=10)
                except subprocess.TimeoutExpired:
                    logging.error(f"killing worker which didn't exit after {on_exit.name}")
                    p.send_signal(signal.SIGKILL)
        return p.wait()


class PIDClient:
    def __init__(self, addr: Union[str, int, Tuple[str, int]]) -> None:
        self.addr = addr
        self.sock = None  # type: Optional[socket.socket]

    def start(self) -> "PIDClient":
        if self.sock is not None:
            return self
        try:
            if isinstance(self.addr, str):
                # Unix socket.
                self.sock = socket.socket(family=socket.AF_UNIX)
                self.sock.connect(self.addr)
            elif isinstance(self.addr, int):
                self.sock = socket.socket()
                self.sock.connect(("127.0.0.1", self.addr))
            else:
                # A address and a port.
                self.sock = socket.socket()
                self.sock.connect(self.addr)
            # Send our PID to the PIDServer.
            self.sock.send(b"%d\n" % os.getpid())
            return self
        except Exception:
            self.close(graceful=False)
            raise

    def close(self, graceful: bool) -> None:
        if self.sock:
            if graceful:
                try:
                    self.sock.send(b"q")
                except Exception:
                    pass
            self.sock.close()
            self.sock = None

    def __enter__(self) -> "PIDClient":
        return self.start()

    def __exit__(self, e_type: type, e_val: Exception, _: Any) -> None:
        # A "graceful" exit is either no exception at all, or a sys.exit(0).
        self.close(graceful=e_type is None or isinstance(e_val, SystemExit) and e_val.code == 0)

    def keep_alive(self) -> None:
        assert self.sock, "must be started first"
        self.sock.send(b"k")

    def run_subprocess(self, cmd: List[str]) -> int:
        p = subprocess.Popen(cmd)

        while True:
            try:
                return p.wait(timeout=60)
            except subprocess.TimeoutExpired:
                self.keep_alive()
