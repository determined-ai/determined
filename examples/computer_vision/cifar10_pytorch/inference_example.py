import logging
import os
import pathlib
import shutil

import filelock

# import chromadb
import torch
import torchvision

# from datasets import load_dataset
from torchvision import transforms

from determined import pytorch
from determined.experimental import client
from determined.pytorch import experimental

from .model_def import download_dataset

FROG_LABEL = 6


# Simple example Inference Processor that demonstrates how to associate
# generic inference metrics with a model version
class FrogCountingInferenceProcessor(experimental.TorchBatchProcessor):
    def __init__(self, context):
        self.context = context
        self.core_context = context.core_context

        hparams = self.context.get_hparams()

        model = client.get_model(hparams.get("model_name"))
        model_version = model.get_version(hparams.get("model_version"))
        self.core_context.utils.report_task_using_model_version(model_version)

        path = model_version.checkpoint.download()
        training_trial = pytorch.load_trial_from_checkpoint_path(
            path, torch_load_kwargs={"map_location": torch.device("cpu")}
        )
        self.model = context.prepare_model_for_inference(training_trial.model)

        self.device = context.device
        self.rank = self.context.distributed.get_rank()

        self.total_frogs = {}
        for rank in range(self.context.distributed.get_size()):
            self.total_frogs[rank] = 0

    def process_batch(self, batch, batch_idx) -> None:
        model_input, labels = batch
        model_input = self.context.to_device(model_input)
        with torch.no_grad():
            pred = self.model(model_input)
            _, predicted = torch.max(pred.data, 1)
            for i in range(len(labels)):
                if predicted[i] == FROG_LABEL:
                    self.total_frogs[self.rank] += 1
        self.last_index = batch_idx

    def on_finish(self):
        self.core_context.train.report_metrics(
            group="inference",
            steps_completed=self.rank,
            metrics={
                "total_frogs": self.total_frogs[self.rank],
            },
        )


if __name__ == "__main__":
    dataset = download_dataset(train=False)
    experimental.torch_batch_process(
        FrogCountingInferenceProcessor,
        dataset,
        batch_size=64,
        checkpoint_interval=10,
    )
