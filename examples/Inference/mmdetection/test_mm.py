import pathlib

import mmcv
import torch

from mmdet.apis import init_detector, inference_detector
from open_image_dataset import OpenImageDataset
from torch.profiler import ProfilerActivity

from determined.experimental.inference import TorchBatchProcessor, torch_batch_process


class MyProcessor(TorchBatchProcessor):
    def __init__(self, context):
        # mmdetection's init_detector already set model to eval and set device
        config_file = "/mmdetection/configs/faster_rcnn/faster_rcnn_r50_caffe_fpn_1x_coco.py"
        cfg = mmcv.Config.fromfile(config_file)
        model = init_detector(cfg, device=context.get_device())
        self.model = model
        self.torch_profiler = torch.profiler.profile(
            activities=[ProfilerActivity.CPU, ProfilerActivity.CUDA],
            schedule=torch.profiler.schedule(wait=1, warmup=1, active=2, repeat=2),
            on_trace_ready=torch.profiler.tensorboard_trace_handler(context.get_tensorboard_path()),
        )
        self.context = context

    def process_batch(self, batch, batch_idx) -> None:
        for img_path in batch:
            with self.torch_profiler as p:
                image = mmcv.imread(img_path)
                # Torch.no_grad is set within inference detector
                pred = inference_detector(self.model, image)
                p.step()

        file_name = f"prediction_output_{batch_idx}"

        with self.context.get_default_storage_path() as path:
            file_path = pathlib.PosixPath(path, file_name)
            output = {"predictions": pred, "input": batch}
            torch.save(output, file_path)


if __name__ == "__main__":
    data = OpenImageDataset("/run/determined/workdir/shared_fs/test_images")
    torch_batch_process(
        MyProcessor,
        data,
        batch_size=1,
    )
