import collections
import json
import logging
import os
import pathlib
import random
import time
from typing import Any, Dict, Optional

import numpy as np

import determined as det
from determined import horovod, profiler, workload
from determined.common import check


class NoOpTrialController(det.CallbackTrialController):
    """
    A trial class which does nothing (except for maybe sleep) during training
    and validation.  For testing purposes.
    """

    CHECKPOINT_FILENAME = "no_op_checkpoint"

    def __init__(
        self,
        prof: profiler.ProfilerAgent,
        context: det.TrialContext,
        env: det.EnvContext,
        workloads: workload.Stream,
        load_path: Optional[pathlib.Path],
        rendezvous_info: det.RendezvousInfo,
        hvd_config: horovod.HorovodContext,
    ) -> None:
        super().__init__(
            context=context,
            env=env,
            workloads=workloads,
            load_path=load_path,
            rendezvous_info=rendezvous_info,
            hvd_config=hvd_config,
            prof=prof,
        )

        check_startup_hook_ran = self.env.hparams.get("check_startup_hook_ran", False)
        if check_startup_hook_ran:
            check.true(os.path.isfile("startup-hook-ran"), "File should exists.")

        self.chaos = random.SystemRandom()
        self._batch_size = self.context.get_per_slot_batch_size()
        self.chaos_probability = self.env.hparams.get("chaos_probability", 0)
        self.chaos_probability_train = self.env.hparams.get("chaos_probability_train")
        self.chaos_probability_validate = self.env.hparams.get("chaos_probability_validate")
        self.chaos_probability_checkpoint = self.env.hparams.get("chaos_probability_checkpoint")
        self.fail_on_first_validation = self.env.hparams.get("fail_on_first_validation", "")
        self.fail_on_chechpoint_save = self.env.hparams.get("fail_on_chechpoint_save", "")
        self.validation_set_size = self.env.hparams.get("validation_set_size", 32 * 32)
        self.train_batch_secs = self.env.hparams.get("training_batch_seconds", 0)
        self.validation_secs = self.env.hparams.get(
            "validation_seconds",
            self.validation_set_size * self.train_batch_secs / self._batch_size,
        )
        self.num_training_metrics = self.env.hparams.get("num_training_metrics", 1)
        assert self.num_training_metrics > 0
        self.num_validation_metrics = self.env.hparams.get("num_validation_metrics", 1)
        assert self.num_validation_metrics > 0
        self.save_secs = self.env.hparams.get("save_checkpoint_seconds", 0)
        self.load_secs = self.env.hparams.get("load_checkpoint_secs", 0)
        self.metrics_progression = self.env.hparams.get("metrics_progression", "decreasing")
        assert self.metrics_progression in ("increasing", "decreasing", "constant")
        self.metrics_base = self.env.hparams.get("metrics_base", 0.9)
        assert 0 < self.metrics_base < 1
        self.metrics_sigma = self.env.hparams.get("metrics_sigma", 0.0)
        assert 0 <= self.metrics_sigma
        self.write_null = self.env.hparams.get("write_null", False)

        self.request_stop = self.env.hparams.get("request_stop", False)

        if self.load_path is None:
            self.trained_steps = collections.Counter()
        else:
            self.load(self.load_path)

    @staticmethod
    def from_trial(
        trial_inst: "det.Trial",
        prof: profiler.ProfilerAgent,
        context: det.TrialContext,
        env: det.EnvContext,
        workloads: workload.Stream,
        load_path: Optional[pathlib.Path],
        rendezvous_info: det.RendezvousInfo,
        hvd_config: horovod.HorovodContext,
    ) -> det.TrialController:
        return NoOpTrialController(
            context=context,
            env=env,
            workloads=workloads,
            load_path=load_path,
            rendezvous_info=rendezvous_info,
            hvd_config=hvd_config,
            prof=prof,
        )

    @staticmethod
    def pre_execute_hook(env: det.EnvContext, hvd_config: horovod.HorovodContext) -> None:
        np.random.seed(env.trial_seed)

    def steps_trained(self) -> int:
        return sum(self.trained_steps.values())

    def current_metric(self) -> float:
        noise = np.random.normal(loc=0.0, scale=self.metrics_sigma ** 2)
        if self.metrics_progression == "constant":
            return self.metrics_base + noise
        elif self.metrics_progression == "decreasing":
            return self.metrics_base ** self.steps_trained() + noise
        elif self.metrics_progression == "increasing":
            return 1 - (self.metrics_base ** self.steps_trained()) + noise
        else:
            raise ValueError("Invalid `metrics_progression` {}".format(self.metrics_progression))

    def train_for_step(self, step_id: int, num_batches: int) -> Dict[str, Any]:
        if self.request_stop:
            self.context.set_stop_requested(True)
        self.chaos_failure(self.chaos_probability_train)
        time.sleep(self.train_batch_secs * num_batches)
        if self.write_null:
            with open("/dev/stdout", "wb") as f:
                f.write(b"\x00")
        self.trained_steps[step_id] += 1
        metrics = {name: self.current_metric() for name in ["loss", *self.training_metrics()]}
        response = {
            "metrics": det.util.make_metrics(
                self._batch_size * num_batches, [metrics] * num_batches
            ),
            "stop_requested": self.context.get_stop_requested(),
        }
        return response

    def compute_validation_metrics(self, step_id: int) -> Dict[str, Any]:
        if self.fail_on_first_validation:
            raise Exception(self.fail_on_first_validation)
        self.chaos_failure(self.chaos_probability_validate)
        time.sleep(self.validation_secs)
        metrics = {
            name: self.current_metric() for name in ["validation_error", *self.validation_metrics()]
        }
        response = {
            "metrics": {"validation_metrics": metrics, "num_inputs": self.validation_set_size},
            "stop_requested": self.context.get_stop_requested(),
        }
        return response

    def training_metrics(self) -> Dict[str, Any]:
        return {"metric_{}".format(i): None for i in range(1, self.num_training_metrics)}

    def validation_metrics(self) -> Dict[str, Any]:
        return {
            "validation_metric_{}".format(i): None for i in range(1, self.num_validation_metrics)
        }

    def batch_size(self) -> int:
        return self._batch_size

    def save(self, path: pathlib.Path) -> None:
        if self.fail_on_chechpoint_save:
            raise Exception(self.fail_on_chechpoint_save)
        self.chaos_failure(self.chaos_probability_checkpoint)
        time.sleep(self.save_secs)
        if not path.exists():
            path.mkdir(parents=True, exist_ok=True)
        fpath = path.joinpath(self.CHECKPOINT_FILENAME)
        logging.info("Saving checkpoint {}, steps_trained {}".format(fpath, self.steps_trained()))
        with fpath.open("w") as f:
            json.dump(self.trained_steps, f, sort_keys=True, indent=4)
        path.chmod(0o777)
        fpath.chmod(0o777)

    def load(self, path: pathlib.Path) -> None:
        self.chaos_failure(self.chaos_probability_checkpoint)
        time.sleep(self.load_secs)
        fpath = path.joinpath(self.CHECKPOINT_FILENAME)
        with fpath.open("r") as f:
            jbody = {int(k): v for k, v in json.load(f).items()}
            for k, v in jbody.items():
                check.gt_eq(k, 0)
                check.is_type(v, int)
                check.gt_eq(v, 0)
            self.trained_steps = collections.Counter(jbody)
            logging.info(
                "Loaded checkpoint {}, steps_trained {}".format(fpath, self.steps_trained())
            )

    def chaos_failure(self, probability: Optional[float]) -> None:
        if probability is None:
            probability = self.chaos_probability
        if self.chaos.random() < probability:
            raise Exception("CHAOS! Executing random failure.")


class NoOpTrial(det.Trial):
    trial_controller_class = NoOpTrialController

    def __init__(self, context: det.TrialContext) -> None:
        self.context = context
