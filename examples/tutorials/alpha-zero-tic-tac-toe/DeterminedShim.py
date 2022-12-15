import determined as det
import pathlib
import logging
import atexit
import sys

from typing import Any, Dict
from utils import *

# Change this to DEBUG to see more info.
log_level = logging.INFO
log = logging.getLogger(__name__)


class DeterminedShim:
    def __init__(self):
        self.steps = -1
        self.info = det.get_cluster_info()
        self.context = det.core.init().__enter__()
        self.storage = None
        self.store_path = None
        self.restore = None
        self.restore_path = None
        self.restore_meta = None
        self._logging_setup()
        self._stage_restore()

    def __del__(self):
        self._release("storage")
        self._release("restore")
        self._release("context")
        log.info("Determined context exited")

    def _logging_setup(self):
        if self.info is None:
            import coloredlogs

            coloredlogs.install(level=log_level)
        else:
            logging.basicConfig(level=log_level, format=det.LOG_FORMAT)

    def _release(self, attr):
        ctx = getattr(self, attr)
        if ctx is not None:
            ctx.__exit__(None, None, None)
            setattr(self, attr, None)

    def _latest_checkpoint(self):
        if self.info is None:
            return None
        if self.info.latest_checkpoint is not None:
            return self.info.latest_checkpoint
        if "latest_checkpoint" in self.info.user_data:
            return self.info.user_data["latest_checkpoint"]
        return None

    def _stage_restore(self, required=True):
        latest_checkpoint = self._latest_checkpoint()
        if latest_checkpoint is None:
            return
        try:
            self.restore = self.context.checkpoint.restore_path(latest_checkpoint)
            self.restore_path = self.restore.__enter__()
            self.restore_meta = self.context.checkpoint.get_metadata(latest_checkpoint)
            if "steps_completed" in self.restore_meta:
                self.steps = self.restore_meta["steps_completed"]
            log.info("checkpoint.get_metadata(%s) -> %r", latest_checkpoint, self.restore_meta)
        except Exception as err:
            log.error("unable to load latest_checkpoint: %r", err)
            if required:
                raise err

    def step(self, metadata=None):
        self.steps += 1
        if self.info is None:
            return
        self._release("storage")
        if self.context.preempt and self.context.preempt.should_preempt():
            log.warn("task early exiting on Determined preempt")
            sys.exit(0)
        if metadata is None:
            metadata = {}
        if not "steps_completed" in metadata:
            metadata["steps_completed"] = self.steps
        self.storage = self.context.checkpoint.store_path(metadata)
        (self.store_path, _storage_id) = self.storage.__enter__()
        log.info("step: checkpoint.store_path(%r) -> %s", metadata, self.store_path)

    def save_path(self, folder):
        if self.store_path is None:
            return pathlib.Path(folder)
        return self.store_path.joinpath(folder)

    def load_path(self, folder):
        if self.restore_path is None:
            return (pathlib.Path(folder), {})
        return (self.restore_path.joinpath(folder), self.restore_meta)

    def training_metrics(self, history: Dict[str, Any]):
        if self.context.train is None:
            return
        batch_metrics = []
        for key, items in history.items():
            for i, val in enumerate(items):
                if i >= len(batch_metrics):
                    batch_metrics.insert(i, {})
                batch_metrics[i][key] = val
        self.context.train.report_training_metrics(self.steps, batch_metrics[-1], batch_metrics)

    def validation_metrics(self, metrics):
        if self.context.train is None:
            return
        self.context.train.report_validation_metrics(self.steps, metrics)

    # help: need stable way to get max_length from core api
    def max_length(self, default):
        if self.info is None or self.info.trial is None:
            return default
        max_len = self.info.trial._config["searcher"]["max_length"]
        log.info("max_length(default: %s) -> %s", default, max_len)
        return max_len

    def override_params(self, params, folder=None, name=None):
        json = {}
        if folder is not None and name is not None:
            (folder, _meta) = self.load_path(folder)
            json = read_json(folder, name)

        def _override(name, source, target):
            for key in source.keys() & target.keys():
                target[key] = source[key]
                log.info("assigned %s parameter: %s = %s", name, key, source[key])

        _override("json", json, params)
        if self.info is not None:
            _override("user", self.info.user_data, params)
            if self.info.trial is not None:
                _override("hyper", self.info.trial.hparams, params)
        return params


shim = DeterminedShim()
atexit.register(shim.__del__)
