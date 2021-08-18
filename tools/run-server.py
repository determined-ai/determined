import argparse
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
        ["../master/build/determined-master", "--config-file", "master.yaml"],
        logs_handler=lambda line: f"{MAGENTA}determined-master  |{CLEAR} {line}",
    )


def run_agent(id: int, slots: int, resource_pool: str) -> mp.Process:
    container_master_host = "host.docker.internal" if sys.platform == "darwin" else ""
    return proc(
        "agent",
        [
            "../agent/build/determined-agent",
            "run",
            "--config-file",
            "agent.yaml",
            "--container-master-host",
            container_master_host,
            "--resource-pool",
            resource_pool,
            "--agent-id",
            f"{id}",
            "--fluent-container-name",
            f"fluent-{id}",
            "--fluent-port",
            f"{5170 + id}",
            "--artificial-slots",
            f"{slots}",
        ],
        logs_handler=lambda line: f"{BLUE}determined-agent-{id}   |{CLEAR} {line}",
    )


def is_db_running() -> bool:
    try:
        with socket.create_connection(("localhost", 5432), timeout=0.5):
            return True
    except OSError:
        return False


def main() -> None:
    parser = argparse.ArgumentParser(description="Process some integers.")
    parser.add_argument("--agents", type=int, default=1)
    parser.add_argument("--slots-per-agent", type=int, default=1)
    args = parser.parse_args()

    db, master, agents, db_logs = False, None, [], None
    try:
        master = run_master()
        for i in range(args.agents):
            agents.append(run_agent(i, args.slots_per_agent, "compute-pool"))
        agents.append(run_agent(i, 0, "aux-pool"))
        db_logs = tail_db_logs()
        if not is_db_running():
            db = True
            subprocess.check_call(["docker-compose", "up", "-d"])

        wait_for_server(5432)
        db_logs.start()
        master.start()
        wait_for_server(8080)
        for i in range(args.agents):
            agent = agents[i]
            agent.start()

        # Join the agent first so we can exit if the agent fails to connect to
        # the master.
        for i in range(args.agents):
            agent = agents[i]
            agent.join()
            if agent.exitcode != 0:
                raise Exception(
                    f"agent {i} failed with non-zero exit code {agent.exitcode}"
                )

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
