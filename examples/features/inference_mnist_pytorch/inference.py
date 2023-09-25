import os
from typing import Any

import filelock
import torch
from torchvision import datasets, transforms

from determined import pytorch
from determined.experimental import client
from determined.pytorch import experimental


# Simple example Inference Processor that demonstrates how to associate
# generic inference metrics with a model version
class ExampleInferenceProcessor(experimental.TorchBatchProcessor):
    def __init__(self, context):
        self.context = context

        hparams = self.context.get_hparams()

        model = client.get_model(hparams.get("model_name"))
        model_version = model.get_version(hparams.get("model_version"))
        self.context.report_task_using_model_version(model_version)

        path = model_version.checkpoint.download()
        training_trial = pytorch.load_trial_from_checkpoint_path(
            path, torch_load_kwargs={"map_location": torch.device("cpu")}
        )
        self.model = context.prepare_model_for_inference(training_trial.model)

        self.device = context.device
        self.rank = self.context.get_distributed_rank()

        self.counts = {}
        for rank in range(self.context.get_distributed_size()):
            self.counts[rank] = {"total_correct": 0, "total": 0}

    def process_batch(self, batch, batch_idx) -> None:
        model_input, labels = batch
        model_input = self.context.to_device(model_input)
        with torch.no_grad():
            pred = self.model(model_input)
            _, predicted = torch.max(pred.data, 1)
            for i in range(len(labels)):
                if predicted[i] == 9:
                    self.counts[self.rank]["num_nines"] += 1
            self.counts[self.rank]["total"] += len(labels)
        self.last_index = batch_idx

    def on_finish(self):
        self.context.report_metrics(
            group="inference",
            steps_completed=self.rank,
            metrics={
                "nine_ratio": self.counts[self.rank]["num_nines"]
                / float(self.counts[self.rank]["total"]),
            },
        )


def download_dataset(train: bool) -> Any:
    download_directory = "data"
    os.makedirs(download_directory, exist_ok=True)
    # Use a file lock so that workers on the same node attempt the download one at a time.
    # The first worker will actually perform the download, while the subsequent workers will
    # see that the dataset is downloaded and skip.
    with filelock.FileLock(os.path.join(download_directory, "lock")):
        return datasets.MNIST(
            download_directory,
            train=train,
            transform=transforms.Compose(
                [
                    transforms.ToTensor(),
                    # These are the precomputed mean and standard deviation of the
                    # MNIST data; this normalizes the data to have zero mean and unit
                    # standard deviation.
                    transforms.Normalize((0.1307,), (0.3081,)),
                ]
            ),
            download=True,
        )


if __name__ == "__main__":
    dataset = download_dataset(train=False)
    experimental.torch_batch_process(
        NineRatioInferenceProcessor,
        dataset,
        batch_size=64,
        checkpoint_interval=10,
    )
