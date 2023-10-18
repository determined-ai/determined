import io
import json
import logging
import os
import shutil
import signal
import socket
import ssl
import subprocess
import sys
import tempfile
import textwrap
import threading
import time
from typing import Any, List, Optional

import pytest

here = os.path.dirname(__file__)
static_srv = os.path.join(here, "../../../master/static/srv")
old = sys.path
try:
    sys.path = [static_srv] + sys.path
    import ship_logs
finally:
    sys.path = old


class ShipLogServer:
    """
    A tiny, hacky http(s) server for testing ship logs.

    It's about the same amount of code as subclassing the stdlib's SimpleHTTPRequestHandler, but it
    shuts down much faster.
    """

    def __init__(self, ctx: Optional[ssl.SSLContext] = None, reject_logs: bool = False) -> None:
        self.ctx = ctx
        self.reject_logs = reject_logs
        self.quit = False
        self.logs: List[str] = []

        self.listener = socket.socket()
        self.listener.bind(("127.0.0.1", 0))
        self.listener.listen(5)

        _, self.port = self.listener.getsockname()

        self.thread = threading.Thread(target=self.serve_requests)
        self.thread.start()

    def __enter__(self) -> "ShipLogServer":
        return self

    def __exit__(self, *args: Any) -> None:
        self.quit = True
        # Wake up the accept() call.
        try:
            with socket.socket() as s:
                s.connect(("127.0.0.1", self.port))
                s.send(b"quit")
        except Exception:
            logging.error("failed to wake up accept loop", exc_info=True)
        self.thread.join()
        self.listener.close()

    def serve_requests(self) -> None:
        try:
            while not self.quit:
                # Accept a conneciton.
                s, _ = self.listener.accept()
                try:
                    if self.quit:
                        return
                    if self.ctx:
                        s = self.ctx.wrap_socket(s, server_side=True)
                    try:
                        self.serve_one_request(s)
                    except Exception:
                        logging.error("error reading request", exc_info=True)
                finally:
                    s.close()
        except Exception:
            logging.error("server crashed", exc_info=True)

    def serve_one_request(self, s: socket.socket) -> None:
        # Receive headers.
        hdrs = b""
        while b"\r\n\r\n" not in hdrs:
            buf = s.recv(4096)
            if not buf:
                # EOF
                return
            hdrs += buf
        # Detect the initial GET /api/v1/me probe.
        if hdrs.startswith(b"GET"):
            s.sendall(b"HTTP/1.1 200 OK\r\n\r\n")
            return
        # Are we supposed to misbehave?
        if self.reject_logs:
            s.sendall(b"HTTP/1.1 500 No! I don't wanna!\r\n\r\n")
            return
        # Receive body until we have valid json.
        hdrs, body = hdrs.split(b"\r\n\r\n", maxsplit=1)
        while True:
            try:
                jbody = json.loads(body)
                break
            except json.decoder.JSONDecodeError:
                # Must not have the full body yet.
                pass
            buf = s.recv(4096)
            if not buf:
                # EOF
                return
            body += buf

        # Remember the logs we saw.
        self.logs.extend(j["log"] for j in jbody["logs"])

        # Send a response.
        s.sendall(b"HTTP/1.1 200 OK\r\n\r\n")

    def master_url(self) -> str:
        return f"http://127.0.0.1:{self.port}"


def mkcmd(script: str) -> List[str]:
    # -u: don't buffer stdout/stderr.
    return [sys.executable, "-u", "-c", textwrap.dedent(script)]


class TestShipLogs:
    """
    A suite of unit tests for ship_logs.py

    Yeah, it's a hack that these tests live in e2e tests.  But since they test python code it's
    just easier this way.
    """

    def run_ship_logs(
        self,
        master_url: str,
        cmd: List[str],
        log_wait_time: float = 30,
        cert_name: str = "",
        cert_file: str = "",
    ) -> int:
        exit_code = ship_logs.main(
            master_url=master_url,
            cert_name=cert_name,
            cert_file=cert_file,
            metadata={},
            token="token",
            emit_stdout_logs=False,
            cmd=cmd,
            log_wait_time=log_wait_time,
        )
        assert isinstance(exit_code, int), exit_code
        return exit_code

    @pytest.mark.e2e_cpu
    def test_exit_code_is_preserved(self) -> None:
        cmd = mkcmd(
            """
            import sys
            print("hi", file=sys.stdout, flush=True)
            print("bye", file=sys.stderr, flush=True)
            sys.exit(9)
            """
        )
        with ShipLogServer() as srv:
            exit_code = self.run_ship_logs(srv.master_url(), cmd)
        assert exit_code == 9, exit_code
        # Ordering of stdout vs stderr is non-deterministic.
        assert set(srv.logs) == {"hi\n", "bye\n"}, srv.logs

    @pytest.mark.e2e_cpu
    def test_cr_to_lf(self) -> None:
        cmd = mkcmd(
            r"""
            print("1\n", end="")
            print("2\r", end="")
            print("3\r\n", end="")
            """
        )
        with ShipLogServer() as srv:
            exit_code = self.run_ship_logs(srv.master_url(), cmd)
        assert exit_code == 0, exit_code
        assert "".join(srv.logs) == "1\n2\n3\n\n", srv.logs

    @pytest.mark.e2e_cpu
    def test_line_endings_are_kept(self) -> None:
        # If re.DOTALL is not used in the regex patterns, then pattern matching can unintentionally
        # strip the newlines out of the end of the line.
        cmd = mkcmd(
            r"""
            print("match neither\n", end="")
            print("[rank=0] match just rank\n", end="")
            print("[rank=0] INFO: match rank and level\n", end="")
            print("INFO: match just level\n", end="")
            """
        )
        with ShipLogServer() as srv:
            exit_code = self.run_ship_logs(srv.master_url(), cmd)
        assert exit_code == 0, exit_code
        expect_logs = [
            "match neither\n",
            "match just rank\n",
            "match rank and level\n",
            "match just level\n",
        ]
        assert srv.logs == expect_logs, srv.logs

    @pytest.mark.e2e_cpu
    def test_stdout_stderr_ordering(self) -> None:
        # Stdout and stderr are collected on different threads, and therefore _can't_ be perfectly
        # synced.  But they should be "approximately" synced; i.e. each 1-second batch should
        # contain both log types.
        #
        # Most dev machines probably will be fine with small timeouts, but CI machines might be
        # slower and we allow up to 0.2 seconds of slop.
        timeouts = [0.001, 0.2]
        for timeout in timeouts:
            cmd = mkcmd(
                f"""
                import sys
                import time
                print("1", file=sys.stdout, flush=True)
                time.sleep({timeout})
                print("2", file=sys.stderr, flush=True)
                time.sleep({timeout})
                print("3", file=sys.stdout, flush=True)
                time.sleep({timeout})
                print("4", file=sys.stderr, flush=True)
                time.sleep({timeout})
                print("5", file=sys.stdout, flush=True)
                time.sleep({timeout})
                print("6", file=sys.stderr, flush=True)
                """
            )
            with ShipLogServer() as srv:
                exit_code = self.run_ship_logs(srv.master_url(), cmd)
            assert exit_code == 0, exit_code
            if "".join(srv.logs) == "1\n2\n3\n4\n5\n6\n":
                # Success
                break
            elif timeout == timeouts[-1]:
                # Failed, even on the highest timeout
                raise ValueError("".join(srv.logs))

    @pytest.mark.e2e_cpu
    def test_signals_are_forwarded(self) -> None:
        cmd = mkcmd(
            """
            import signal
            import time

            def handle_sigint(*arg):
                print("caught sigint!")

            signal.signal(signal.SIGINT, handle_sigint)

            print("ready!", flush=True)

            time.sleep(5)
            """
        )
        with ShipLogServer() as srv:
            # Start a subprocess so we can signal it.
            env = {
                "DET_MASTER": srv.master_url(),
                "DET_SESSION_TOKEN": "token",
                "DET_TASK_LOGGING_METADATA": "{}",
                "DET_SHIPPER_EMIT_STDOUT_LOGS": "1",
            }
            fullcmd = [sys.executable, "-u", ship_logs.__file__] + cmd
            p = subprocess.Popen(fullcmd, env=env, stdout=subprocess.PIPE)
            assert p.stdout
            try:
                # Wait for the granchild log to indicate signals are set up.
                for line in p.stdout:
                    if b"ready!" in line:
                        break
                # Send a signal that is caught and logged, to test signal forwarding.
                p.send_signal(signal.SIGINT)
                for line in p.stdout:
                    if b"caught sigint!" in line:
                        break
                # Send a signal that is not caught, to test for signal exit codes.
                p.send_signal(signal.SIGTERM)
                exit_code = p.wait()
            finally:
                p.kill()
                p.wait()
        assert exit_code == 128 + signal.SIGTERM, exit_code
        assert "".join(srv.logs) == "ready!\ncaught sigint!\n", srv.logs

    @pytest.mark.e2e_cpu
    def test_exit_wait_time_is_honored(self) -> None:
        cmd = mkcmd("print('hello world')")
        # We configure the server to reject logs in order to guarantee the shipper times out.
        with ShipLogServer(reject_logs=True) as srv:
            start = time.time()
            exit_code = self.run_ship_logs(srv.master_url(), cmd, log_wait_time=0.1)
            end = time.time()
            assert exit_code == 0, exit_code
            assert end - start < 1, end - start

    @pytest.mark.e2e_cpu
    def test_entrypoint_not_found_exits_127(self) -> None:
        cmd = ["/does-not-exist"]
        with ShipLogServer() as srv:
            exit_code = self.run_ship_logs(srv.master_url(), cmd)
        # 127 is the standard bash exit code for file-not-found.
        assert exit_code == 127, exit_code
        assert "FileNotFoundError" in "".join(srv.logs), srv.logs

    @pytest.mark.e2e_cpu
    def test_entrypoint_not_executable_exits_126(self) -> None:
        cmd = ["/bin/"]
        with ShipLogServer() as srv:
            exit_code = self.run_ship_logs(srv.master_url(), cmd)
        # 126 is the standard bash exit code for permission failure.
        assert exit_code == 126, exit_code
        assert "PermissionError" in "".join(srv.logs), srv.logs

    @pytest.mark.e2e_cpu
    def test_only_standard_library_dependences(self) -> None:
        cmd = mkcmd(
            """
            # ONLY STANDARD LIBRARY IMPORTS ARE ALLOWED
            import datetime
            import io
            import json
            import logging
            import os
            import queue
            import re
            import signal
            import ssl
            import subprocess
            import sys
            import threading
            import time
            import traceback
            import typing
            import urllib.request
            # END OF STANDARD LIBRARY IMPORTS

            # Now the only new module that `import ship_logs` can add is ship_logs itself.
            allowed_modules = set((*sys.modules, "ship_logs"))

            sys.path = ["%s"] + sys.path
            import ship_logs

            new_modules = set(sys.modules).difference(allowed_modules)

            for name in new_modules:
                print("possible non-standard-library dependency detected:", name)

            exit(1 if new_modules else 0)
            """
            % (static_srv)
        )
        p = subprocess.Popen(cmd, stdout=subprocess.PIPE)
        assert p.stdout
        errmsgs = p.stdout.read().decode("utf8")
        assert p.wait() == 0, "\n" + errmsgs

    @pytest.mark.e2e_cpu
    @pytest.mark.parametrize("noverify", (True, False))
    def test_custom_certs(self, noverify: bool) -> None:
        # Use the untrusted key and cert from the harness unit tests.
        untrusted = os.path.join(here, "../../../harness/tests/common/untrusted-root")
        keyfile = os.path.join(untrusted, "127.0.0.1-key.pem")
        certfile = os.path.join(untrusted, "127.0.0.1-cert.pem")

        # Create the server ssl context.
        ctx = ssl.create_default_context(purpose=ssl.Purpose.CLIENT_AUTH)
        ctx.load_cert_chain(certfile=certfile, keyfile=keyfile)

        cmd = mkcmd("print('hello world')")

        with ShipLogServer(ctx) as srv:
            # Use the wrong name to talk to the server, to test name verification override.
            master_url = f"https://localhost:{srv.port}"
            exit_code = self.run_ship_logs(
                master_url,
                cmd,
                cert_file="noverify" if noverify else certfile,
                cert_name="127.0.0.1",
            )
            assert exit_code == 0, exit_code
            assert srv.logs == ["hello world\n"], srv.logs

    @pytest.mark.e2e_cpu
    def test_honor_http_proxy(self) -> None:
        # Use a subprocess to control the environment.
        with ShipLogServer() as srv:
            env = {
                "DET_MASTER": "http://notreal.faketld",
                "DET_SESSION_TOKEN": "token",
                "DET_TASK_LOGGING_METADATA": "{}",
                "http_proxy": srv.master_url(),
            }
            cmd = mkcmd("print('hello world')")
            fullcmd = [sys.executable, "-u", ship_logs.__file__] + cmd
            subprocess.run(fullcmd, env=env, check=True)
            assert srv.logs == ["hello world\n"], srv.logs

    @pytest.mark.e2e_cpu
    def test_honor_no_proxy(self) -> None:
        # Use a subprocess to control the environment.
        with ShipLogServer() as srv:
            env = {
                "DET_MASTER": srv.master_url(),
                "DET_SESSION_TOKEN": "token",
                "DET_TASK_LOGGING_METADATA": "{}",
                "http_proxy": "http://notreal.faketld",
                "NO_PROXY": "127.0.0.1",
            }
            cmd = mkcmd("print('hello world')")
            fullcmd = [sys.executable, "-u", ship_logs.__file__] + cmd
            subprocess.run(fullcmd, env=env, check=True)
            assert srv.logs == ["hello world\n"], srv.logs

    @pytest.mark.e2e_cpu
    def test_escape_hatch(self) -> None:
        # Create a temporary directory to catch our escape-hatch logs
        tmp = tempfile.mkdtemp(suffix="ship_logs")
        try:
            with ShipLogServer() as srv:
                # Use a subprocess to control the environment.
                # Leave out DET_MASTER to force a crash.
                env = {"DET_SHIP_LOGS_PATH": tmp}
                cmd = mkcmd("pass")
                fullcmd = [sys.executable, "-u", ship_logs.__file__] + cmd
                p = subprocess.run(fullcmd, env=env)
                assert p.returncode == 80, p.returncode
                assert srv.logs == [], srv.logs
                files = os.listdir(tmp)
                assert len(files) == 2, files
                assert "ship-logs-ran" in files, files
                files.remove("ship-logs-ran")
                with open(os.path.join(tmp, files[0]), "r") as f:
                    text = f.read()
                assert "KeyError: 'DET_MASTER'" in text, text
        finally:
            shutil.rmtree(tmp)


class TestReadNewlinesOrCarriageReturns:
    # read_newlines_or_carriage_returns is designed to read from filedescriptors resulting from
    # subprocess.Popen(bufsize=0).stdout, and different kinds of filehandles can result in slightly
    # different read/write behaviors, and stdout and stderr can additionally be different on
    # different operating systems, so this test will use Popen to create file descriptors instead of
    # something more convenient like os.pipe(), in order to test against the most realistic
    # conditions.

    @pytest.mark.e2e_cpu
    @pytest.mark.parametrize("lastline", [repr("hi"), repr("hi\r"), repr("hi\n")])
    def test_eof_handling_after_process_killed(self, lastline: str) -> None:
        cmd = mkcmd(
            r"""
            import sys
            import time
            print(%s, end="", flush=True)
            # message test to kill us, then wait
            print(".", file=sys.stderr, flush=True)
            time.sleep(10)
            """
            % lastline
        )
        start = time.time()
        p = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, bufsize=0)
        assert p.stdout and p.stderr
        reader = ship_logs.read_newlines_or_carriage_returns(p.stdout)
        # wait for message on stderr
        _ = p.stderr.read(1)
        p.kill()
        p.wait()
        line = next(reader)
        end = time.time()
        assert line == "hi\n", line
        # Make sure the test didn't wait for that sleep(10) to finish.
        assert end - start < 1, end - start

    @pytest.mark.e2e_cpu
    @pytest.mark.parametrize("lastline", ["hi", "hi\r", "hi\n"])
    def test_eof_handling_after_process_closes_stdout(self, lastline: str) -> None:
        cmd = mkcmd(
            r"""
            import sys
            import time
            import os
            print(%s, end="", flush=True)
            # close stdout
            os.close(1)
            # message test that it's done, then wait
            print(".", file=sys.stderr, flush=True)
            time.sleep(10)
            """
            % repr(lastline)
        )
        start = time.time()
        p = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, bufsize=0)
        assert p.stdout and p.stderr
        reader = ship_logs.read_newlines_or_carriage_returns(p.stdout)
        # wait for message on stderr
        _ = p.stderr.read(1)
        line = next(reader)
        assert line == "hi\n", line
        # stdout is now empty
        with pytest.raises(StopIteration):
            next(reader)
        p.kill()
        p.wait()
        end = time.time()
        # Make sure the test didn't wait for that sleep(10) to finish.
        assert end - start < 1, end - start

    @pytest.mark.e2e_cpu
    def test_long_lines(self) -> None:
        s = "abcdefghijklmopqrstuvwxyz"
        # Reader will buffer up to io.DEAULT_BUFFER_SIZE-1 before forcing a line break.
        max_chars = io.DEFAULT_BUFFER_SIZE - 1
        n = (max_chars + len(s) - 1) // len(s)
        msg = s * n
        exp_1 = msg[:max_chars] + "\n"
        exp_2 = msg[max_chars:] + "\n"
        cmd = mkcmd("print(%s, flush=True)" % repr(msg))
        p = subprocess.Popen(cmd, stdout=subprocess.PIPE, bufsize=0)
        assert p.stdout
        reader = ship_logs.read_newlines_or_carriage_returns(p.stdout)
        line = next(reader)
        assert line == exp_1, line
        line = next(reader)
        assert line == exp_2, line
        p.wait()
