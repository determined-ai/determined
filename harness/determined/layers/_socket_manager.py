import logging
import socket
import ssl
from typing import Any, Optional

import lomond
import lomond.session
import simplejson

import determined as det
from determined import layers, util, workload


class CustomSSLWebsocketSession(lomond.session.WebsocketSession):  # type: ignore
    """
    A session class that allows for the TLS verification mode of a WebSocket connection to be
    configured based on values from the context.
    """

    def __init__(self, socket: lomond.WebSocket, env: det.EnvContext) -> None:
        super().__init__(socket)
        self.ctx = ssl.SSLContext(ssl.PROTOCOL_TLSv1_2)
        self.ctx.verify_mode = ssl.CERT_REQUIRED
        self.ctx.check_hostname = True
        if env.master_cert_file is not None:
            self.ctx.load_verify_locations(cafile=env.master_cert_file)
        else:
            self.ctx.load_default_certs()

    def _wrap_socket(self, sock: socket.SocketType, host: str) -> socket.SocketType:
        return self.ctx.wrap_socket(sock, server_hostname=host)


class SocketManager(workload.Source):
    """
    SocketManager handles WebSocket-related events common to any harness.
    Workload messages, which may vary in type depending on the task being run,
    are passed to the WorkloadManager.
    """

    def __init__(self, env: det.EnvContext) -> None:
        self.env = env

        url = "{}://{}:{}/ws/trial/{}/{}/{}".format(
            "wss" if self.env.use_tls else "ws",
            self.env.master_addr,
            self.env.master_port,
            self.env.initial_workload.experiment_id,
            self.env.initial_workload.trial_id,
            self.env.container_id,
        )

        # Disable reading proxy configuration because we shouldn't proxy our
        # own connection to the master.
        self.socket = lomond.WebSocket(url, proxies={})

        self.ws_events = self.socket.connect(
            ping_rate=0, session_class=lambda socket: CustomSSLWebsocketSession(socket, env)
        )

        # Handle the messages up to and including the rendezvous message.
        for ws_event in self.ws_events:
            ri = self.check_for_rendezvous_info(ws_event)
            if ri is None:
                continue
            self.rendezvous_info = ri
            break
        else:
            raise ValueError("Ran out of events without finding rendezvous message")

    def __iter__(self) -> workload.Stream:
        # Always yield the initial workload first.
        yield from self.yield_workload(self.env.initial_workload)

        # Then pass workloads which arrive on the websocket.
        for ws_event in self.ws_events:
            yield from self.handle_event(ws_event)

    def __enter__(self) -> "SocketManager":
        return self

    def __exit__(self, *_: Any) -> None:
        self.close()

    def close(self) -> None:
        self.socket.close()

        # Empty the websocket.
        for ws_event in self.ws_events:
            if not self.message_is_log_only(ws_event):
                logging.warning(f"Unexpected websocket event: {ws_event}")

    def get_rendezvous_info(self) -> det.RendezvousInfo:
        return self.rendezvous_info

    def message_is_log_only(self, event: Any) -> bool:
        if isinstance(event, lomond.events.Connecting):
            logging.info("Connecting to master at %s", event.url)
        elif isinstance(event, lomond.events.Connected):
            logging.info("Connected to master")
        elif isinstance(event, lomond.events.ConnectFail):
            logging.warning("Failed to connect to master: %s", event.reason)
        elif isinstance(event, lomond.events.Closing):
            logging.info("Server started WebSocket shutdown: %s", event.reason)
        elif isinstance(event, lomond.events.Closed):
            logging.info("WebSocket closed" + (f": {event.reason}" if event.reason else ""))
        elif isinstance(event, lomond.events.Disconnected):
            # The event loop will exit after this event is received.
            if event.graceful:
                logging.info("Disconnected from master, exiting gracefully")
            else:
                logging.warning("Disconnected from master abruptly: %s", event.reason)
        elif isinstance(event, lomond.events.ProtocolError):
            logging.warning("WebSocket protocol error: %s", event.error)
        elif isinstance(event, lomond.events.Rejected):
            logging.warning("Master rejected WebSocket: %s", event.reason)
        elif isinstance(event, lomond.events.Ready):
            logging.info("Established WebSocket session with master")
        elif isinstance(event, lomond.events.Poll):
            logging.debug("WebSocket poll")
        else:
            return False
        return True

    def check_for_rendezvous_info(self, event: Any) -> Optional[det.RendezvousInfo]:
        """
        Wait for a message from the socket, and check if it is a det.RendezvousInfo.

        Raise an error if a Workload is seen, since those should only come after
        det.RendezvousInfo.
        """

        if self.message_is_log_only(event):
            return None
        elif isinstance(event, lomond.events.Text):
            msg = simplejson.loads(event.text)

            if msg["type"] == "RENDEZVOUS_INFO":
                logging.info("Got rendezvous information: %s", msg)

                # If there's only one container, there's nothing to do for
                # rendezvous.
                addrs, rank = msg["addrs"], msg["rank"]
                addrs2 = msg["addrs2"]

                # The rendezvous info contains the external addresses for
                # all the containers, but we need to set what to actually
                # bind to inside this container. We just bind to the
                # wildcard interface, on a fixed port that matches the one
                # the agent is hardcoded to expose in all trial containers.
                # TODO(DET-916): Make number of ports configurable.
                rendezvous_ports = self.env.rendezvous_ports()
                addrs[rank] = f"0.0.0.0:{rendezvous_ports[0]}"
                addrs2[rank] = f"0.0.0.0:{rendezvous_ports[1]}"

                return det.RendezvousInfo(addrs, addrs2, rank)

            elif msg["type"] == "RUN_WORKLOAD":
                raise ValueError("Received workload before rendezvous info")
        else:
            logging.warning(f"unexpected websocket event: {event}")

        return None

    def handle_event(self, event: Any) -> workload.Stream:
        if self.message_is_log_only(event):
            return
        elif isinstance(event, lomond.events.Text):
            msg = simplejson.loads(event.text)
            if msg["type"] == "RUN_WORKLOAD":
                wkld = workload.Workload.from_json(msg["workload"])
                yield from self.yield_workload(wkld)
            else:
                raise NotImplementedError(f"Unrecognized message: {msg}")
        else:
            logging.warning(f"Unexpected websocket event: {event}")

    def yield_workload(self, wkld: workload.Workload) -> workload.Stream:
        if self.env.debug:
            logging.debug("Starting profiler...")
            profiler = layers.HarnessProfiler(use_gpu=self.env.use_gpu)
            profiler.start()

        # When the workload manager responds, forward the message to the master.
        def respond(metrics: workload.Response) -> None:

            # Close the socket after the response from the TERMINATE message, in case the trial
            # wasn't smart enough to exit after the TERMINATE on its own.
            if wkld.kind == workload.Workload.Kind.TERMINATE:
                self.close()
                return

            # Handle skipped workloads gracefully.
            if isinstance(metrics, workload.Skipped):
                return

            duration = metrics["end_time"] - metrics["start_time"]
            logging.info(f"Workload completed: {metrics['workload']} (duration {duration})")

            self.socket.send_text(util.json_encode(metrics))

        yield wkld, [], respond

        if self.env.debug:
            profiler.stop()
            profiler.serialize_raw_results(f"/tmp/step-{wkld.step_id}-{wkld.kind}.json")
            profiler.serialize_graph(f"/tmp/step-{wkld.step_id}-{wkld.kind}.png")
