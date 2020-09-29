import multiprocessing as mp
import socket
import subprocess
import sys
import time
from typing import Callable, List, Optional

# While we could use something like requests (or any other 3rd-party module),
# this script aims to work with the default Python 3.6+.

CLEAR = "\033[39m"
MAGENTA = "\033[95m"
BLUE = "\033[94m"

DB_PROT = 5433
MASTER_PORT = 8081

def kill_process(name: str, process: Optional[mp.Process]) -> None:
    if process is not None and process.is_alive():
        try:
            process.terminate()
        except Exception:
            print(f"failed to kill process: {name}")


def wait_for_server(port: int, host: str = "localhost", timeout: float = 5.0) -> None:
    for _ in range(100):
        try:
            with socket.create_connection((host, port), timeout=timeout):
                return
        except OSError:
            time.sleep(1)
    print(f"Timed out waiting for the {host}:{port}.")


def proc(name: str, cmd: List[str], logs_handler: Callable = lambda x: x) -> mp.Process:
    def func() -> None:
        with subprocess.Popen(
            cmd,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
        ) as p:
            try:
                assert p.stdout is not None
                for line in p.stdout:
                    print(logs_handler(line.decode("utf8")), end="", flush=True)
            except KeyboardInterrupt:
                print(f"Killing Log stream for {name}")

    return mp.Process(target=func, daemon=True)


def tail_db_logs() -> mp.Process:
    return proc("database-logs", ["docker-compose", "logs", "-f"])


def run_master() -> mp.Process:
    return proc(
        "master",
        ["../../../master/build/determined-master", "--config-file", "master.yaml"],
        logs_handler=lambda line: f"{MAGENTA}determined-master  |{CLEAR} {line}",
    )


def run_agent() -> mp.Process:
    container_master_host = "host.docker.internal" if sys.platform == "darwin" else ""
    return proc(
        "agent",
        [
            "../../../agent/build/determined-agent",
            "run",
            "--config-file",
            "agent.yaml",
            "--container-master-host",
            container_master_host,
        ],
        logs_handler=lambda line: f"{BLUE}determined-agent   |{CLEAR} {line}",
    )


def is_db_running() -> bool:
    try:
        with socket.create_connection(("localhost", DB_PROT), timeout=0.5):
            return True
    except OSError:
        return False


def main() -> None:
    db, master, agent, db_logs = False, None, None, None
    try:
        master = run_master()
        agent = run_agent()
        db_logs = tail_db_logs()
        if not is_db_running():
            db = True
            subprocess.check_call(["docker-compose", "up", "-d"])

        wait_for_server(DB_PROT)
        db_logs.start()
        master.start()
        wait_for_server(MASTER_PORT)
        agent.start()

        # Join the agent first so we can exit if the agent fails to connect to
        # the master.
        agent.join()
        if agent.exitcode != 0:
            raise Exception(f"agent failed with non-zero exit code {agent.exitcode}")

        master.join()
        db_logs.join()
    except KeyboardInterrupt:
        pass
    finally:
        kill_process("master", master)
        kill_process("agent", agent)
        kill_process("db-logs", db_logs)
        if db:
            subprocess.check_call(["docker-compose", "down"])


if __name__ == "__main__":
    main()
