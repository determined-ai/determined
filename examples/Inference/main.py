import torch

import torchvision as tv
import torchvision.transforms as transforms

from _torch_offline_distributed_dataset import (
    TorchPerBatchProcessor,
    TorchDistributedDatasetProcessor,
    initialize_distributed_backend,
)
from model import get_model

import determined as det

import pathlib

from torch.profiler import ProfilerActivity


class MyProcessor(TorchPerBatchProcessor):
    def __init__(self, model):
        self.model = model
        self.device = "cuda" if torch.cuda.is_available() else "cpu"
        model.eval()
        model.to(self.device)

    def process_batch(self, batch, additional_info) -> None:
        model_input = batch[0]
        model_input = model_input.to(self.device)
        with torch.no_grad():
            with additional_info.torch_profiler as p:
                pred = self.model(model_input)
                p.step()
        file_name = f"prediction_output_{additional_info.batch_idx}_{additional_info.worker_rank}"
        file_path = pathlib.PosixPath(
            "/run/determined/workdir/shared_fs/new_runner_inference_out", file_name
        )
        output = {"predictions": pred, "input": batch}
        torch.save(output, file_path)


if __name__ == "__main__":
    with det.core.init(distributed=initialize_distributed_backend()) as core_context:
        transform = transforms.Compose(
            [transforms.ToTensor(), transforms.Normalize((0.5, 0.5, 0.5), (0.5, 0.5, 0.5))]
        )
        inference_data = tv.datasets.CIFAR10(
            root=".\\data", train=False, download=True, transform=transform
        )

        model = get_model()

        predictor = TorchDistributedDatasetProcessor(
            core_context, MyProcessor(model), inference_data, batch_size=64
        )

        predictor.set_torch_profiler(
            activities=[ProfilerActivity.CPU, ProfilerActivity.CUDA],
            schedule=torch.profiler.schedule(wait=1, warmup=1, active=2, repeat=2),
        )

        predictor.run()
