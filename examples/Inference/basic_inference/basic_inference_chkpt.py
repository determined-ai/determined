import os
import pathlib

import filelock
import torch
import torchvision as tv
import torchvision.transforms as transforms
from model import get_model
from torch.profiler import ProfilerActivity

from determined.experimental.inference import TorchBatchProcessor, torch_batch_process


class MyProcessor(TorchBatchProcessor):
    def __init__(self, context):
        self.context = context
        self.model = context.prepare_model_for_inference(get_model())

        self.profiler = torch.profiler.profile(
            activities=[ProfilerActivity.CPU, ProfilerActivity.CUDA],
            schedule=torch.profiler.schedule(wait=1, warmup=1, active=2, repeat=2),
            on_trace_ready=torch.profiler.tensorboard_trace_handler(context.get_tensorboard_path()),
        )
        self.output = []
        self.last_index = 0

    def process_batch(self, batch, batch_idx) -> None:
        model_input = batch[0]
        model_input = self.context.to_device(model_input)

        with torch.no_grad():
            with self.profiler as p:
                pred = self.model(model_input)
                p.step()
                output = {"predictions": pred, "input": batch}
                self.output.append(output)

        self.last_index = batch_idx

    def run_before_checkpoint(self):
        if len(self.output) == 0:
            return
        file_name = f"prediction_output_{self.last_index}"
        with self.context.get_default_storage_path() as path:
            file_path = pathlib.PosixPath(path, file_name)
            torch.save(self.output, file_path)
        self.output = []


if __name__ == "__main__":
    transform = transforms.Compose(
        [transforms.ToTensor(), transforms.Normalize((0.5, 0.5, 0.5), (0.5, 0.5, 0.5))]
    )
    with filelock.FileLock(os.path.join("/tmp", "inference.lock")):
        inference_data = tv.datasets.CIFAR10(
            root="/data", train=False, download=True, transform=transform
        )
    torch_batch_process(
        MyProcessor,
        inference_data,
        batch_size=64,
    )
