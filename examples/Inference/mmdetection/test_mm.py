import pathlib


import torch
from torch.profiler import ProfilerActivity

from _torch_offline_distributed_dataset import (
    TorchPerBatchProcessor,
    TorchDistributedDatasetProcessor,
    initialize_distributed_backend,
)

import determined as det
from open_image_dataset import OpenImageDataset

import mmcv
from mmdet.apis import init_detector, inference_detector


class MyProcessor(TorchPerBatchProcessor):
    def __init__(self, model):
        self.model = model
        self.device = "cuda" if torch.cuda.is_available() else "cpu"
        model.eval()
        model.to(self.device)

    def process_batch(self, batch, additional_info) -> None:
        for img_path in batch:
            with torch.no_grad():
                with additional_info.torch_profiler as p:
                    image = mmcv.imread(img_path)
                    pred = inference_detector(self.model, image)
                    p.step()

        file_name = f"prediction_output_{additional_info.batch_idx}_{additional_info.worker_rank}"
        file_path = pathlib.PosixPath(
            "/run/determined/workdir/shared_fs/new_runner_inference_out/mmdetection", file_name
        )
        output = {"predictions": pred, "input": batch}
        torch.save(output, file_path)


if __name__ == "__main__":
    with det.core.init(distributed=initialize_distributed_backend()) as core_context:
        data = OpenImageDataset("/run/determined/workdir/shared_fs/open_images/small_test")

        config_file = "/mmdetection/configs/faster_rcnn/faster_rcnn_r50_caffe_fpn_1x_coco.py"

        cfg = mmcv.Config.fromfile(config_file)
        model = init_detector(cfg)

        predictor = TorchDistributedDatasetProcessor(
            core_context, MyProcessor(model), data, batch_size=2
        )

        predictor.set_torch_profiler(
            activities=[ProfilerActivity.CPU, ProfilerActivity.CUDA],
            schedule=torch.profiler.schedule(wait=1, warmup=1, active=2, repeat=2),
        )

        predictor.run()
