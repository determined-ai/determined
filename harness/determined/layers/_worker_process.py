import pathlib
import pickle
from typing import cast

import determined as det
from determined import horovod
from determined.common import check


class WorkerProcessContext:
    def __init__(
        self,
        hvd_config: horovod.HorovodContext,
        rendezvous_info: det.RendezvousInfo,
        env: det.EnvContext,
    ) -> None:
        self.hvd_config = hvd_config
        self.rendezvous_info = rendezvous_info
        self.env = env

    @staticmethod
    def from_file(path: pathlib.Path) -> "WorkerProcessContext":
        with path.open(mode="rb") as f:
            obj = pickle.load(f)
        check.is_instance(obj, WorkerProcessContext, "did not find WorkerProcessContext in file")
        return cast(WorkerProcessContext, obj)

    def to_file(self, path: pathlib.Path) -> None:
        with path.open(mode="wb") as f:
            pickle.dump(self, f)
