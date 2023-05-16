import torch

import torchvision as tv
import torchvision.transforms as transforms

from _torch_batch_process import (
    get_default_device,
    initialize_default_inference_context,
    TorchPerBatchProcessor,
    torch_batch_process,
)
from model import get_model

import pathlib

from torch.profiler import ProfilerActivity


class MyProcessor(TorchPerBatchProcessor):
    def __init__(self, model, profiler, device):
        self.model = model
        self.device = device
        model.eval()
        model.to(self.device)
        self.profiler = profiler

    def process_batch(self, batch, additional_info) -> None:
        model_input = batch[0]
        model_input = model_input.to(self.device)
        with torch.no_grad():
            with self.profiler as p:
                pred = self.model(model_input)
                p.step()

        file_name = f"prediction_output_{additional_info.batch_idx}_{additional_info.worker_rank}"
        file_path = pathlib.PosixPath(
            "/run/determined/workdir/shared_fs/new_runner_inference_out", file_name
        )
        output = {"predictions": pred, "input": batch}
        torch.save(output, file_path)


if __name__ == "__main__":
    with initialize_default_inference_context() as core_context:
        transform = transforms.Compose(
            [transforms.ToTensor(), transforms.Normalize((0.5, 0.5, 0.5), (0.5, 0.5, 0.5))]
        )
        inference_data = tv.datasets.CIFAR10(
            root=".\\data", train=False, download=True, transform=transform
        )

        model = get_model()

        torch_profiler = torch.profiler.profile(
            activities=[ProfilerActivity.CPU, ProfilerActivity.CUDA],
            schedule=torch.profiler.schedule(wait=1, warmup=1, active=2, repeat=2),
            on_trace_ready=torch.profiler.tensorboard_trace_handler(
                str(core_context.train.get_tensorboard_path())
            ),
        )

        torch_batch_process(
            core_context,
            MyProcessor(model, torch_profiler, get_default_device(core_context)),
            inference_data,
            batch_size=64,
        )
