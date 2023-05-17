import pathlib

import filelock

import os

import deepspeed
import torch
import torchvision as tv
import torchvision.transforms as transforms
from torch.profiler import ProfilerActivity

import _torch_batch_process as batch
import determined as det

from model import get_model

dtype = torch.float16

train_batch_size = 2

ds_config = {
    "train_batch_size": 128,
    "steps_per_print": 10,
    "optimizer": {
        "type": "Adam",
        "params": {"lr": 0.001, "betas": [0.8, 0.999], "eps": 1e-8, "weight_decay": 3e-7},
    },
    "scheduler": {
        "type": "WarmupLR",
        "params": {"warmup_min_lr": 0, "warmup_max_lr": 0.001, "warmup_num_steps": 1000},
    },
    "zero_optimization": {
        "stage": 3,
        "offload_optimizer": {
            "device": "cpu",
            "pin_memory": True,
            "buffer_count": 4,
            "fast_init": False,
        },
        "offload_param": {
            "device": "cpu",
            "pin_memory": True,
            "buffer_count": 5,
            "buffer_size": 1e8,
            "max_in_cpu": 1e9,
        },
        "allgather_partitions": True,
        "allgather_bucket_size": 5e8,
        "overlap_comm": True,
        "reduce_scatter": True,
        "reduce_bucket_size": 5e8,
        "contiguous_gradients": True,
        "stage3_max_live_parameters": 1e9,
        "stage3_max_reuse_distance": 1e9,
        "stage3_prefetch_bucket_size": 5e8,
        "stage3_param_persistence_threshold": 1e6,
    },
    "gradient_clipping": 1.0,
    "fp16": {
        "enabled": True,
        "loss_scale": 0,
        "initial_scale_power": 5,
        "loss_scale_window": 1000,
        "hysteresis": 2,
        "min_loss_scale": 1,
    },
}


class MyProcessor(batch.TorchBatchProcessor):
    def __init__(self, core_context, init_info):
        device = init_info.default_device
        tensorboard_path = init_info.tensorboard_path

        with deepspeed.zero.Init():
            model = get_model()
            model.to(device)
            model.eval()
        model_engine = deepspeed.initialize(model=model, config=ds_config)[0]
        model_engine.module.eval()
        self.model = model_engine.module
        self.device = device
        self.profiler = torch.profiler.profile(
            activities=[ProfilerActivity.CPU, ProfilerActivity.CUDA],
            schedule=torch.profiler.schedule(wait=1, warmup=1, active=2, repeat=2),
            on_trace_ready=torch.profiler.tensorboard_trace_handler(tensorboard_path),
        )
        self.worker_rank = init_info.worker_rank

    def process_batch(self, batch, batch_idx) -> None:
        model_input = batch[0]
        model_input = model_input.to(self.device)
        model_input = model_input.half()
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


def main():
    transform = transforms.Compose(
        [transforms.ToTensor(), transforms.Normalize((0.5, 0.5, 0.5), (0.5, 0.5, 0.5))]
    )

    with filelock.FileLock(os.path.join("/tmp", "inference.lock")):
        inference_data = tv.datasets.CIFAR10(
            root="/data", train=False, download=True, transform=transform
        )

    batch.torch_batch_process(MyProcessor, inference_data, batch_size=64, dataloader_drop_last=True)


if __name__ == "__main__":
    main()
