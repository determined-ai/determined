import abc
import pathlib

DEVCLUSTER_LOG_PATH = pathlib.Path("/tmp/devcluster")


class Cluster(metaclass=abc.ABCMeta):
    @abc.abstractmethod
    def __init__(self) -> None:
        pass

    @abc.abstractmethod
    def kill_master(self) -> None:
        pass

    @abc.abstractmethod
    def restart_master(self) -> None:
        pass

    @abc.abstractmethod
    def restart_agent(self, wait_for_amnesia: bool = True, wait_for_agent: bool = True) -> None:
        pass

    @abc.abstractmethod
    def ensure_agent_ok(self) -> None:
        pass

    def log_marker(self, marker: str) -> None:
        for log_path in DEVCLUSTER_LOG_PATH.glob("*.log"):
            with log_path.open("a") as fout:
                fout.write(marker)
