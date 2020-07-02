import logging
import time
from typing import Any, Callable, Dict, List, Optional, Tuple, cast

import zmq
from zmq.error import ZMQBindError, ZMQError

from determined_common import check


class _OneSidedBarrier:
    """
    _OneSidedBarrier is a message from participants (usually workers) to a single process (usually
    the chief) indicating to the chief that the workers are ready for the next phase of
    computation.
    """

    def __init__(self, message: Any) -> None:
        self.message = message


class MetricsInfo:
    """
    MetricsInfo contains validation metrics and the number of batches used to generate those
    metrics. Used to communicate metrics between training processes.
    """

    def __init__(self, metrics: Dict[str, Any], num_batches: int):
        self.metrics = metrics
        self.num_batches = num_batches


class ConnectedMessage:
    """
    ConnectedMessage is sent by a ZMQBroadcastClient to a ZMQBroadcastServer as the very first
    message. The ZMQBroadcastServer must gather one ConnectedMessage from each client before it is
    safe to broadcast.
    """

    pass


class ReadyMessage:
    """
    ReadyMessage is sent by a SubprocessReceiver to the SubprocessLauncher when it is ready to
    start receiving workloads.
    """

    pass


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
    """

    def __init__(
        self, num_connections: int, pub_port: Optional[int] = None, pull_port: Optional[int] = None
    ) -> None:
        self._num_connections = num_connections

        context = zmq.Context()

        self._pub_socket = context.socket(zmq.PUB)
        self._pull_socket = context.socket(zmq.PULL)

        if pub_port is None:
            self._pub_port = self._pub_socket.bind_to_random_port("tcp://*")  # type: int
        else:
            self._pub_port = self._pub_socket.bind(f"tcp://*:{pub_port}")

        if pull_port is None:
            self._pull_port = self._pull_socket.bind_to_random_port("tcp://*")  # type: int
        else:
            self._pull_port = self._pull_socket.bind(f"tcp://*:{pull_port}")

        self._send_serial = 0
        self._recv_serial = 0

    def __enter__(self) -> "ZMQBroadcastServer":
        return self

    def __exit__(self, *_: Any) -> None:
        self.close()

    def close(self) -> None:
        self._pub_socket.close()
        self._pull_socket.close()

    def get_pub_port(self) -> int:
        return self._pub_port

    def get_pull_port(self) -> int:
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
        context = zmq.Context()

        self._sub_socket = context.socket(zmq.SUB)
        # Subscriber always listens to ALL messages.
        self._sub_socket.subscribe(b"")
        self._sub_socket.connect(srv_pub_url)

        self._push_socket = context.socket(zmq.PUSH)
        self._push_socket.connect(srv_pull_url)

        self._send_serial = 0
        self._recv_serial = 0

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
        self.context = zmq.Context()
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
            except ZMQError:
                logging.warning(f"Failed to bind to port {port}.")
                exit(1)
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
            except ZMQBindError:
                logging.warning(f"Failed to bind to port range {port_range}.")
                exit(1)
            self.sockets.append(socket)

    def get_ports(self) -> List[int]:
        return self.ports

    def send(self, py_obj: Any) -> None:
        for socket in self.sockets:
            socket.send_pyobj(py_obj)

    def receive_blocking(self, send_rank: int) -> Any:
        check.lt(send_rank, len(self.sockets))
        message = self.sockets[send_rank].recv_pyobj()
        return message

    def receive_non_blocking(
        self, send_rank: int, deadline: Optional[float] = None
    ) -> Tuple[bool, Any]:
        check.lt(send_rank, len(self.sockets))
        timeout = 1000 if not deadline else int(deadline - time.time()) * 1000
        timeout = max(timeout, 100)

        if self.sockets[send_rank].poll(timeout) == 0:
            return False, None
        message = self.sockets[send_rank].recv_pyobj()
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
            self.sockets[0].send_pyobj(_OneSidedBarrier(message=message))

        return messages

    def close(self) -> None:
        for socket in self.sockets:
            socket.close()


class ZMQClient:
    """
    ZMQClient connects training subprocesses with trial-controller.
    It also signals the chief trial-controller, when the non-chief
    trial controller has successfully started sshd.
    """

    def __init__(self, ip_address: str, port: int) -> None:
        self.context = zmq.Context()
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
