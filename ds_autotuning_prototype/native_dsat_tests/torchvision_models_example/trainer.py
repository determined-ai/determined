import argparse
import gc
import json
import logging
import os
import pathlib
import random
import shutil
from typing import Callable, Generator, List, Literal, Tuple, Union

import data
import deepspeed
import determined as det
import numpy as np
import torch
import torch.nn as nn
import utils
from constants import DS_CONFIG_PATH, FLOPS_PROFILER_OUTPUT_PATH
from determined.pytorch import TorchData


class DeepSpeedTrainer(nn.Module):
    def __init__(
        self,
        core_context: det.core.Context,
        args: argparse.Namespace,
        model: nn.Module,
        train_dataset: torch.utils.data.Dataset,
        random_seed: int = 42,
    ) -> None:
        super().__init__()
        self.core_context = core_context
        self.args = args
        self.model = model
        self.train_dataset = train_dataset
        self.random_seed = random_seed

        self.criterion = nn.CrossEntropyLoss()

        self.rank = core_context.distributed.rank
        self.is_chief = self.rank == 0
        self.local_rank = core_context.distributed.local_rank
        self.is_local_chief = self.local_rank == 0

        self.steps_completed = 0

        # Instantiated as needed through private methods.
        self.train_loader = None
        self.model_engine = None
        self.optimizer = None
        self.fp16 = None
        self.device = None

        self._deepspeed_init()

    def _set_random_seeds(self) -> None:
        random.seed(self.random_seed)
        np.random.seed(self.random_seed)
        torch.random.manual_seed(self.random_seed)

    def _deepspeed_init(self) -> None:
        deepspeed.init_distributed()
        self.model_engine, self.optimizer, self.train_loader, __ = deepspeed.initialize(
            args=self.args,
            model=self.model,
            model_parameters=self.model.parameters(),
            training_data=self.train_dataset,
        )
        self.fp16 = self.model_engine.fp16_enabled()
        # DeepSpeed uses the local_rank as the device, for some reason.
        self.device = self.model_engine.device

    def _batch_generator(
        self,
    ) -> Generator[Tuple[torch.Tensor, torch.Tensor], None, None]:
        for batch in self.train_loader:
            inputs, targets = batch
            if self.fp16:
                inputs = inputs.half()
            inputs = inputs.to(self.device)
            targets = targets.to(self.device)
            yield inputs, targets

    def _train_one_batch(self, inputs: TorchData, targets: TorchData) -> None:
        outputs = self.model_engine(inputs)
        loss = self.criterion(outputs, targets)
        self.model_engine.backward(loss)
        self.model_engine.step()

    def train_for_step(self) -> None:
        """Train for one SGD step, accounting for GAS."""
        for inputs, targets in self._batch_generator(split="train"):
            self._train_one_batch(inputs=inputs, targets=targets)
            if self.model_engine.is_gradient_accumulation_boundary():
                break

    def train_on_cluster(self) -> None:
        # A single op of fixed length is emitted.
        for op in self.core_context.searcher.operations():
            while self.steps_completed < op.length:
                self.train_for_step()
                self.steps_completed += 1
                if self.core_context.preempt.should_preempt():
                    return
            if self.is_chief:
                # Report completed value is not needed.
                op.report_completed(0)

    def autotuning(self) -> None:
        # A single op of fixed length is emitted.
        for op in self.core_context.searcher.operations():
            while self.steps_completed < op.length:
                self.train_for_step()
                self.steps_completed += 1
                if self.core_context.preempt.should_preempt():
                    return
            if self.is_chief:
                # Report completed value is not needed.
                op.report_completed(0)
        logging.warning("Saving autotuning results.")
        if self.is_chief:
            self._report_and_save_native_autotuning_results()

    def _report_and_save_native_autotuning_results(
        self, path: pathlib.Path = pathlib.Path(".")
    ) -> None:
        results = utils.DSAutotuningResults(path=path)
        ranked_results_dicts = results.get_ranked_results_dicts()
        for rank, results_dict in enumerate(ranked_results_dicts):
            metrics = results_dict["metrics"]
            ds_config = results_dict["exp_config"]["ds_config"]
            reported_metrics = utils.get_flattened_dict({**metrics, **ds_config})
            self.core_context.train.report_validation_metrics(
                steps_completed=rank,
                metrics=reported_metrics,
            )

        checkpoint_metadata_dict = {"steps_completed": len(ranked_results_dicts) - 1}
        with self.core_context.checkpoint.store_path(checkpoint_metadata_dict) as (
            ckpt_path,
            storage_id,
        ):
            for autotuning_dir in ("autotuning_exps", "autotuning_results"):
                src_path = pathlib.Path(autotuning_dir)
                shutil.copytree(
                    src=src_path,
                    dst=pathlib.Path(ckpt_path).joinpath(autotuning_dir),
                )
