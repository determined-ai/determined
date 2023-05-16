import pathlib

import torch
from torch.profiler import ProfilerActivity

from _torch_offline_distributed_dataset import (
    TorchPerBatchProcessor,
    torch_batch_process,
    initialize_default_inference_context,
    get_default_device,
)

from open_image_dataset import OpenImageDataset

import mmcv
from mmdet.apis import init_detector, inference_detector


class MyProcessor(TorchPerBatchProcessor):
    def __init__(self, model, tensorboard_path):
        # mmdetection's init_detector already set model to eval and set device
        self.model = model
        self.torch_profiler = torch.profiler.profile(
            activities=[ProfilerActivity.CPU, ProfilerActivity.CUDA],
            schedule=torch.profiler.schedule(wait=1, warmup=1, active=2, repeat=2),
            on_trace_ready=torch.profiler.tensorboard_trace_handler(tensorboard_path),
        )

    def process_batch(self, batch, additional_info) -> None:
        for img_path in batch:
            with self.torch_profiler as p:
                image = mmcv.imread(img_path)
                # Torch.no_grad is set within inference detector
                pred = inference_detector(self.model, image)
                p.step()

        file_name = f"prediction_output_{additional_info.batch_idx}_{additional_info.worker_rank}"
        file_path = pathlib.PosixPath(
            "/run/determined/workdir/shared_fs/new_runner_inference_out/mmdetection", file_name
        )
        output = {"predictions": pred, "input": batch}
        torch.save(output, file_path)


if __name__ == "__main__":
    with initialize_default_inference_context() as core_context:
        data = OpenImageDataset("/run/determined/workdir/shared_fs/open_images/small_test")

        config_file = "/mmdetection/configs/faster_rcnn/faster_rcnn_r50_caffe_fpn_1x_coco.py"

        cfg = mmcv.Config.fromfile(config_file)
        model = init_detector(cfg, device=str(get_default_device(core_context)))

        torch_batch_process(
            core_context,
            MyProcessor(model, core_context.train.get_tensorboard_path()),
            data,
            batch_size=2,
        )
