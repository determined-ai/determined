"""
THIS LOG SHIPPER MUST ONLY SHIP LOGS.

It is not allowed to depend on any libraries other than python standard libraries, including our own
determined library.  The reason is that python virtual env errors in user containers are common
scenarios for users to encounter, and we must be able to still ship logs in those scenarios.

The only thing that is allowed to break the log shipper is a misconfigured cluster (if the log
shipper isn't able to connect to the master) or if python isn't installed.

But if you are thinking of making this log shipper do anything other than ship logs, or if you are
thinking of adding any dependencies not inside the standard library, stop.  Let the log shipper
just ship logs; it's too important of a job to mix it with anything else.

---

ship_logs.py: a suitable container entrypoint that ships logs from a child process to the master.

usage: ship_logs.py CMD ARGS...

ship_logs.py will read environment variables set by the master to obtain its configuration.  It
isn't intended to be useful in any non-managed environments.
"""

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
import urllib.request
from typing import Any, Dict, Iterator, List, NamedTuple, Optional, cast

# Duplicated from determined/__init__.py.  It's nice to keep them in sync.
LOG_FORMAT = "%(levelname)s: [%(process)s] %(name)s: %(message)s"


# Example log message given below.
# 2022-05-12 16:32:48,757:gc_checkpoints: [rank=0] INFO: Determined checkpoint GC, ...
# Below regex is used to extract the rank field from the log message.
# Excluding empty spaces this regex matches rank in the above example as [rank=0]
# Using the DOTALL flag means we keep the newline at the end of the pattern.
rank = re.compile(
    r"(?P<space1> ?)\[rank=(?P<rank_id>([0-9]+))\](?P<space2> ?)(?P<log>.*)", flags=re.DOTALL
)
# Below regex is used to extract the message severity from the log message.
# Excluding empty spaces and delimiter(:) this regex matches message severity level in the above
# example as INFO.
# Using the DOTALL flag means we keep the newline at the end of the pattern.
level = re.compile(
    r"(?P<space1> ?)(?P<level>(DEBUG|INFO|WARNING|ERROR|CRITICAL)):(?P<space2> ?)(?P<log>.*)",
    flags=re.DOTALL,
)
lineend = re.compile(rb"[\r\n]")


# Interval at which to force a flush.
SHIPPER_FLUSH_INTERVAL = 1  # How often to make API calls

# Full jitter time on encountering an API exception.
SHIPPER_FAILURE_BACKOFF_SECONDS = 1

# Max size of the log buffer before forcing a flush.
LOG_BATCH_MAX_SIZE = 1000

# Max size of the shipping queue before we start to apply backpressure by blocking sends. We would
# only hit this if we got underwater by three full batches while trying to ship a batch.
SHIP_QUEUE_MAX_SIZE = 3 * LOG_BATCH_MAX_SIZE


class DoneMsg(NamedTuple):
    """
    DoneMsg is what each thread of exectuion puts on the doneq when it finishes.
    """

    who: str
    error: Optional[Exception] = None
    exit_code: Optional[int] = None


def read_newlines_or_carriage_returns(fd: io.RawIOBase) -> Iterator[str]:
    r"""
    Read lines, delineated by either '\n' or '\r.

    Unlike the default io.BufferedReader used in subprocess.Popen(bufsize=-1), we read until we
    encounter either '\n' or \r', and treat that as one line.

    Specifically, io.BufferedReader doesn't handle tqdm progress bar outputs very well; it treats
    all of the '\r' outputs as one enormous line.

    Args:
        fd: an unbuffered stdout or stderr from a subprocess.Popen.

    Yields:
        A series of str, one per line.  Each line always ends with a '\n'.  Each line will be
        broken to length io.DEFAULT_BUFFER_SIZE, even if the underlying io didn't have a linebreak.
    """
    # Ship lines of length of DEFAULT_BUFFER_SIZE, including the terminating newline.
    limit = io.DEFAULT_BUFFER_SIZE - 1
    nread = 0
    chunks: List[bytes] = []

    def oneline():
        nonlocal nread
        nonlocal chunks
        out = b"".join(chunks).decode("utf8")
        chunks = []
        nread = 0
        return out

    while True:
        buf = fd.read(limit - nread)
        if not buf:
            # EOF.
            break

        # Extract all the lines from this buffer.
        while buf:
            m = lineend.search(buf)
            if m is None:
                # No line break here; just append to chunks.
                chunks.append(buf)
                nread += len(buf)
                break

            # Line break found!
            start, end = m.span()
            chunks.append(buf[:start])
            # Even if we matched a '\r', emit a '\n'.
            chunks.append(b"\n")
            yield oneline()
            # keep checking the rest of buf
            buf = buf[end:]

        # Detect if we reached our buffer limit.
        if nread >= limit:
            # Pretend we got a line anyway.
            chunks.append(b"\n")
            yield oneline()

    # One last line, maybe.
    if chunks:
        chunks.append(b"\n")
        yield oneline()


class Collector(threading.Thread):
    """
    Collector is the thread that reads and parses lines from stdout or stderr.

    It will pass structured data to the logq, and will send a message on doneq when it finishes.
    """

    def __init__(
        self,
        fd: io.RawIOBase,
        stdtype: str,
        emit_stdout_logs: bool,
        metadata: Dict[str, Any],
        logq: queue.Queue,
        doneq: queue.Queue,
    ) -> None:
        super().__init__()
        self.fd = fd
        self.stdtype = stdtype
        self.metadata = {"stdtype": self.stdtype, **metadata}
        self.logq = logq
        self.doneq = doneq

        if not emit_stdout_logs:
            self.dup_io = None
        else:
            self.dup_io = sys.stdout if stdtype == "stdout" else sys.stderr

        self.shipper_died = False

    def run(self) -> None:
        try:
            self._run()
        except Exception as e:
            self.doneq.put(DoneMsg(self.stdtype, error=e))
        else:
            self.doneq.put(DoneMsg(self.stdtype, error=None))
        finally:
            self.logq.put(None)

    def _run(self) -> None:
        for line in read_newlines_or_carriage_returns(self.fd):
            # Capture the timestamp as soon as the line is collected.
            now = datetime.datetime.now(datetime.timezone.utc).isoformat()

            if self.dup_io:
                print(line, file=self.dup_io, flush=True, end="")

            if self.shipper_died:
                # Keep draining logs so process doesn't block on stdout or stderr, but don't bother
                # queuing the logs we capture.
                continue

            log: Dict[str, Any] = {"timestamp": now, **self.metadata}

            m = rank.match(line)
            if m:
                try:
                    log["rank_id"] = int(m.group("rank_id"))
                    line = m.group("log")
                except ValueError:
                    pass

            m = level.match(line)
            if m:
                found = m.group("level")
                log["level"] = f"LOG_LEVEL_{found}"
                line = m.group("log")

            log["log"] = line

            self.logq.put(log)


def override_verify_name(ctx: ssl.SSLContext, verify_name: str) -> ssl.SSLContext:
    class VerifyNameOverride:
        def __getattr__(self, name: str, default: Any = None) -> Any:
            return getattr(ctx, name, default)

        def wrap_socket(self, *args, server_hostname=None, **kwargs) -> Any:
            kwargs["server_hostname"] = verify_name
            return ctx.wrap_socket(*args, **kwargs)

    return cast(ssl.SSLContext, VerifyNameOverride())


class Shipper(threading.Thread):
    """
    Shipper reads structured logs from logq and ships them to the determined-master.

    It will send a message on doneq when it finishes.
    """

    def __init__(
        self,
        master_url: str,
        token: str,
        cert_name: str,
        cert_file: str,
        logq: queue.Queue,
        doneq: queue.Queue,
        daemon: bool,
    ) -> None:
        super().__init__(daemon=daemon)
        self.logq = logq
        self.doneq = doneq

        # TODO(rb): Switch to DET_USER_TOKEN when the user token passed into a container isn't
        # limited to expire in 7 days, and then set `Authorization: Bearer $token` here instead.
        self.headers = {"Grpc-Metadata-x-allocation-token": f"Bearer {token}"}

        baseurl = master_url.rstrip("/")
        self.url = f"{baseurl}/api/v1/task/logs"

        self.context = None
        if master_url.startswith("https://"):
            # Create an SSLContext that trusts our DET_MASTER_CERT_FILE, and checks the hostname
            # against the DET_MASTER_CERT_NAME (which may differ from the hostname in the url).
            self.context = ssl.create_default_context()
            if cert_file.lower() == "noverify":
                # Don't check the master's certificate.
                # Presently the master never sets this value for DET_MASTER_CERT_FILE, but we keep
                # this check to be consistent with the CLI behavior.
                self.context.verify_mode = ssl.CERT_NONE
            elif cert_file:
                # Explicitly trust the cert in cert_file.
                self.context.load_verify_locations(cafile=cert_file)
            if cert_name:
                # Override hostname verification
                self.context = override_verify_name(self.context, cert_name)

    def run(self) -> None:
        try:
            self._run()
        except Exception as e:
            self.doneq.put(DoneMsg("shipper", error=e))
        else:
            self.doneq.put(DoneMsg("shipper", error=None))

    def _run(self) -> None:
        eofs = 0
        while eofs < 2:
            logs: List[Dict[str, Any]] = []
            deadline = time.time() + SHIPPER_FLUSH_INTERVAL
            # Pop logs until both collectors close, or we fill up a batch, or we hit the deadline.
            while eofs < 2 and len(logs) < LOG_BATCH_MAX_SIZE:
                now = time.time()
                timeout = deadline - now
                if timeout <= 0:
                    # We are already passed the deadline.
                    break

                try:
                    log = self.logq.get(timeout=timeout)
                except queue.Empty:
                    # We hit the timeout.
                    break

                if log is None:
                    eofs += 1
                    continue

                logs.append(log)

            if not logs:
                continue

            data = json.dumps({"logs": logs}).encode("utf8")

            # Try to ship for about ten minutes.
            backoffs = [0, 1, 5, 10, 15, 15, 15, 15, 15, 15, 15, 60, 60, 60, 60, 60, 60, 60, 60, 60]

            self.ship(data, backoffs)

    def ship(self, data: bytes, backoffs: List[int]) -> None:
        for delay in backoffs:
            time.sleep(delay)
            try:
                req = urllib.request.Request(self.url, data, self.headers, method="POST")
                with urllib.request.urlopen(req, context=self.context) as resp:
                    respbody = resp.read()

                if resp.getcode() != 200:
                    raise RuntimeError(
                        f"ship logs failed: status code {resp.get_code()}, body:\n---\n"
                        + respbody.decode("utf8")
                    )

                # Shipped successfully
                return

            except Exception:
                logging.error("failed to ship logs to master", exc_info=True)
                pass

        raise RuntimeError("failed to connect to master for too long, giving up")

    def ship_special(self, msg: str, metadata: Dict[str, str], emit_stdout_logs: bool) -> None:
        """
        Ship a special message, probably from failing to start the child process.
        """
        now = datetime.datetime.now(datetime.timezone.utc).isoformat()
        logs = []
        # Build a json log line out of each message line.
        if not msg.endswith("\n"):
            msg += "\n"
        for line in msg.splitlines(keepends=True):
            if emit_stdout_logs:
                print(line, end="", file=sys.stderr)
            logs.append(
                {
                    "timestamp": now,
                    "log": line,
                    "level": "ERROR",
                    "stdtype": "stderr",
                    **metadata,
                }
            )

        data = json.dumps({"logs": logs}).encode("utf8")

        # Try to ship for about 30 seconds.
        backoffs = [0, 1, 5, 10, 15]
        self.ship(data, backoffs)


class Waiter(threading.Thread):
    """
    Waiter calls p.wait() on a process, that's it.
    """

    def __init__(self, p: subprocess.Popen, doneq: queue.Queue):
        self.p = p
        self.doneq = doneq
        super().__init__()

    def run(self) -> None:
        try:
            exit_code = self.p.wait()
            self.doneq.put(DoneMsg("waiter", exit_code=exit_code, error=None))
        except Exception as e:
            self.doneq.put(DoneMsg("waiter", exit_code=None, error=e))


def main(
    master_url: str,
    cert_name: str,
    cert_file: str,
    metadata: Dict[str, str],
    token: str,
    emit_stdout_logs: bool,
    cmd: List[str],
    log_wait_time: int,
) -> int:
    logq: queue.Queue = queue.Queue()
    doneq: queue.Queue = queue.Queue()

    waiter_started = False
    stdout_started = False
    stderr_started = False
    shipper_started = False

    # Normally we like structured concurrency; i.e. a function that owns a thread must not exit
    # until that thead has been properly cleaned up.  However, it is important that the log shipper
    # is not allowed to keep a task container alive too long after the child process has exited.  We
    # want to guarantee that we exit about DET_LOG_WAIT_TIME seconds after the child process exits.
    #
    # However, interruping a synchronous HTTP call form urllib is nearly impossible; even if you
    # were to select() until the underlying file descriptor had something to read before calling
    # Request.read(), there are many buffered readers in there and most likely multiple os.read()
    # calls would occur and you'd be blocking anyway.
    #
    # So as an easy workaround, we set daemon=True and just exit the process if it's not done on
    # time.
    shipper = Shipper(master_url, token, cert_name, cert_file, logq, doneq, daemon=True)
    shipper_timed_out = False

    # Start the process or ship a special log message to the master why we couldn't.
    try:
        # Don't rely on Popen's standard line buffering; we want to do our own line buffering.
        p = subprocess.Popen(
            cmd, stdin=subprocess.DEVNULL, stdout=subprocess.PIPE, stderr=subprocess.PIPE, bufsize=0
        )
    except FileNotFoundError:
        shipper.ship_special(f"FileNotFoundError executing {cmd}", metadata, emit_stdout_logs)
        # 127 is the standard bash exit code for file-not-found.
        return 127
    except PermissionError:
        # Unable to read or to execute the command.
        shipper.ship_special(f"PermissionError executing {cmd}", metadata, emit_stdout_logs)
        # 126 is the standard bash exit code for permission failure.
        return 126
    except Exception:
        msg = f"unexpected failure executing {cmd}:\n" + traceback.format_exc()
        shipper.ship_special(msg, metadata, emit_stdout_logs)
        # 80 is the exit code we use to signal "ship_logs.py failed"
        return 80

    # Just for mypy.
    assert isinstance(p.stdout, io.RawIOBase) and isinstance(p.stderr, io.RawIOBase)

    try:
        waiter = Waiter(p, doneq)
        waiter.start()
        waiter_started = True

        # Set up signal forwarding.
        def signal_passthru(signum: Any, frame: Any):
            p.send_signal(signum)

        for sig in [
            signal.SIGINT,
            signal.SIGTERM,
            signal.SIGHUP,
            signal.SIGUSR1,
            signal.SIGUSR2,
            signal.SIGWINCH,
        ]:
            signal.signal(sig, signal_passthru)

        stdout = Collector(p.stdout, "stdout", emit_stdout_logs, metadata, logq, doneq)
        stderr = Collector(p.stderr, "stderr", emit_stdout_logs, metadata, logq, doneq)

        stdout.start()
        stdout_started = True

        stderr.start()
        stderr_started = True

        shipper.start()
        shipper_started = True

        exit_code = None
        deadline: Optional[float] = None
        # Expect 4 messages on the doneq, one for each thread we started.
        for _ in range(4):
            # Wait for an event, possibly with a deadline (if the child process already exited).
            try:
                timeout = None if deadline is None else deadline - time.time()
                if timeout is not None and timeout <= 0:
                    raise queue.Empty()
                donemsg = doneq.get(timeout=timeout)
                assert isinstance(donemsg, DoneMsg)
            except queue.Empty:
                # Deadline is done, just abandon the shipper.
                shipper_timed_out = True
                logging.error(
                    f"waited {log_wait_time} seconds for shipper to finish after child exit; "
                    "giving up now"
                )
                break

            if donemsg.who == "shipper":
                # There's no point in collecting logs after the shipper is gone.
                stdout.shipper_died = True
                stderr.shipper_died = True

            if donemsg.error is not None:
                # Something in our shipping machinery broke.
                raise RuntimeError(
                    f"failure in log shipper; {donemsg.who} thread died"
                ) from donemsg.error

            if donemsg.who == "waiter":
                # After the log shipper exits, the shipping code is on a deadline.
                #
                # Note: we could almost just return exit_code here and let the cleanup and shipping
                # timeout happen in the finally block, but the finally block can't easily detect
                # exceptions that arrive on the doneq, so we stay in this for...range(4) loop until
                # we're confident all threads shut down without error.
                deadline = time.time() + log_wait_time
                exit_code = donemsg.exit_code

        # Mypy doesn't know that we're guaranteed to have an exit_code by now.
        #
        # It's guaranteed because the only scenario where we exit the loop without waiting on all
        # the threads is if the Waiter exited but the Shipper didn't.  And the only case where the
        # Waiter doesn't set the exit_code is in the case that p.wait() threw an error, in which
        # case we would have raised an exception due to a non-None donemsg.error.
        assert exit_code is not None

        # Convert signal exits to standard bash-style exits.
        if exit_code < 0:
            return 128 - exit_code
        return exit_code

    finally:
        # If our logging infrastructure ever crashes, just give up on the child process.
        p.kill()
        if waiter_started:
            waiter.join()
        # After p is dead, the Collectors should run out of input and exit quickly.
        if stdout_started:
            stdout.join()
        if stderr_started:
            stderr.join()
        # After everyone else is dead, the shipper could still be in a retry loop for a long time,
        # so we wait up to DET_LOG_WAIT_TIME seconds for it to exit, then we give up on it.
        if shipper_started and not shipper_timed_out:
            shipper.join(timeout=log_wait_time)
            if shipper.is_alive():
                # The timeout was reached.
                logging.error(
                    f"waited {log_wait_time} seconds for shipper to finish after crash; "
                    "giving up now"
                )


def configure_escape_hatch(dirpath: str) -> None:
    """
    Even if the log shipper goes belly-up, dump logs to a bind-mounted path.

    If the log shipper is failing in production, you obviously can't expect to find those logs in
    task logs, so this allows a user or support person to mount a directory into a container in
    order to find out why the log shipper is broken.
    """

    try:
        hostname = os.environ["DET_AGENT_ID"]
    except Exception:
        try:
            import socket

            hostname = socket.gethostname()
        except Exception:
            hostname = "unknown"
    # You can't run logging.basicConfig() twice so we manually add a second handler at the root
    # logging level.
    fh = logging.FileHandler(
        filename=os.path.join(ship_logs_path, f"{hostname}-{time.time()}.log"),
        # Only create the file if we actually log to it (aka if there's an error).  That way if
        # there's lots of processes not failing, we don't create tons of empty files.
        delay=False,
    )
    fh.setFormatter(logging.Formatter(LOG_FORMAT))
    logging.getLogger().addHandler(fh)

    try:
        # Touch a single file to indicate that the escape hatch is working, so that in debugging
        # someone can distinguish "the escape hatch isn't working" from "ship_logs just isn't
        # hitting any errors".
        with open(os.path.join(ship_logs_path, "ship-logs-ran"), "w"):
            pass
    except Exception:
        pass


if __name__ == "__main__":
    try:
        logging.basicConfig(
            format=LOG_FORMAT,
            stream=sys.stderr,
        )

        ship_logs_path = os.environ.get("DET_SHIP_LOGS_PATH", "/ship_logs")
        if os.path.exists(ship_logs_path):
            configure_escape_hatch(ship_logs_path)

        master_url = os.environ["DET_MASTER"]
        cert_name = os.environ.get("DET_MASTER_CERT_NAME", "")
        cert_file = os.environ.get("DET_MASTER_CERT_FILE", "")
        # TODO(rb): fix DET_USER_TOKEN to support tokens with lifetimes tied to an allocation, and
        # use DET_USER_TOKEN here instead.
        token = os.environ["DET_SESSION_TOKEN"]
        raw_metadata = os.environ["DET_TASK_LOGGING_METADATA"]
        try:
            metadata = json.loads(raw_metadata)
            assert isinstance(metadata, dict)
        except Exception:
            raise ValueError(f"invalid DET_TASK_LOGGING_METADATA: '{raw_metadata}'") from None

        metadata["container_id"] = os.environ.get("DET_CONTAINER_ID", "")
        metadata["agent_id"] = os.environ.get("DET_AGENT_ID", "")

        raw_log_wait_time = os.environ.get("DET_LOG_WAIT_TIME", "30")
        try:
            log_wait_time = int(raw_log_wait_time)
        except Exception:
            raise ValueError(f"invalid DET_LOG_WAIT_TIME: '{raw_log_wait_time}'") from None

        emit_stdout_logs = bool(os.environ.get("DET_SHIPPER_EMIT_STDOUT_LOGS"))

        metadata["source"] = "task"

        exit_code = main(
            master_url,
            cert_name,
            cert_file,
            metadata,
            token,
            emit_stdout_logs,
            cmd=sys.argv[1:],
            log_wait_time=log_wait_time,
        )
    except Exception:
        logging.error("ship_logs.py crashed!", exc_info=True)
        sys.exit(80)

    sys.exit(exit_code)
