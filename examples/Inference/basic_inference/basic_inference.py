import filelock

import pathlib

import os

import torch

from torch.profiler import ProfilerActivity

import torchvision as tv
import torchvision.transforms as transforms

from _torch_batch_process import (
    TorchBatchProcessor,
    torch_batch_process,
)

from model import get_model


class MyProcessor(TorchBatchProcessor):
    def __init__(self, core_context, init_info):
        self.model = get_model()
        self.device = init_info.default_device
        self.model.eval()
        self.model.to(self.device)
        self.profiler = torch_profiler = torch.profiler.profile(
            activities=[ProfilerActivity.CPU, ProfilerActivity.CUDA],
            schedule=torch.profiler.schedule(wait=1, warmup=1, active=2, repeat=2),
            on_trace_ready=torch.profiler.tensorboard_trace_handler(init_info.tensorboard_path),
        )
        self.worker_rank = init_info.worker_rank

    def process_batch(self, batch, batch_idx) -> None:
        model_input = batch[0]
        model_input = model_input.to(self.device)
        with torch.no_grad():
            with self.profiler as p:
                pred = self.model(model_input)
                p.step()

        file_name = f"prediction_output_{batch_idx}_{self.worker_rank}"
        file_path = pathlib.PosixPath(
            "/run/determined/workdir/shared_fs/new_runner_inference_out", file_name
        )
        output = {"predictions": pred, "input": batch}
        torch.save(output, file_path)


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
