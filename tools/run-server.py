import multiprocessing as mp
import subprocess
import time
import urllib.request
import urllib.error
import sys

# While we could use something like requests (or any other 3rd-party module),
# this script aims to work with the default Python 3.6+.

CLEAR = "\033[39m"
MAGENTA = "\033[95m"
BLUE = "\033[94m"


def run_master():
    with subprocess.Popen(
        ["../master/build/determined-master", "--config-file", "master.yaml"],
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
    ) as proc:
        for line in proc.stdout:
            print(
                f"{MAGENTA}determined-master  |{CLEAR} {line.decode('utf8')}",
                end="",
                flush=True,
            )


def wait_for_master():
    for _ in range(100):
        try:
            urllib.request.urlopen("http://localhost:8080")
            return
        except (urllib.error.HTTPError, urllib.error.URLError):
            time.sleep(1)
    print("Timed out waiting for the master.")


def run_agent():
    container_master_host = "host.docker.internal" if sys.platform == "darwin" else ""
    with subprocess.Popen(
        [
            "../agent/build/determined-agent",
            "run",
            "--config-file",
            "agent.yaml",
            "--container-master-host",
            container_master_host,
        ],
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
    ) as proc:
        for line in proc.stdout:
            print(
                f"{BLUE}determined-agent   |{CLEAR} {line.decode('utf8')}",
                end="",
                flush=True,
            )


def main():
    try:
        master = mp.Process(target=run_master, daemon=True)
        agent = mp.Process(target=run_agent, daemon=True)
        master.start()
        wait_for_master()
        agent.start()

        # Join the agent first so we can exit if the agent fails to connect to
        # the master.
        agent.join()
        if agent.exitcode != 0:
            master.terminate()
            sys.exit(agent.exitcode)

        master.join()
    except (KeyboardInterrupt, SystemExit):
        pass


if __name__ == "__main__":
    main()
