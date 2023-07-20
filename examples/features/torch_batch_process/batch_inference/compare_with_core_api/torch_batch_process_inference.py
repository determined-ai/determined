import os
import pathlib

import constants
import filelock
import model
import torch
import torchvision as tv
import torchvision.transforms as transforms

from determined.pytorch import experimental


class MyProcessor(experimental.TorchBatchProcessor):
    def __init__(self, context):
        self.model = context.prepare_model_for_inference(model.build_model())
        self.context = context

    def process_batch(self, batch, batch_idx) -> None:
        output_list = []
        model_input = batch[0]
        model_input = self.context.to_device(model_input)
        file_name = f"prediction_output_{batch_idx}"
        with torch.no_grad():
            pred = self.model(model_input)
            output = {"predictions": pred, "input": batch}
            output_list.append(output)

        # Automatic upload to output to the same storage used by determined checkpoints
        with self.context.upload_path() as path:
            file_path = pathlib.Path(path, file_name)
            torch.save(output_list, file_path)


def main():
    pathlib.Path.mkdir(pathlib.Path(constants.DATA_DIRECTORY), parents=True, exist_ok=True)
    transform = transforms.Compose(
        [transforms.ToTensor(), transforms.Normalize((0.5, 0.5, 0.5), (0.5, 0.5, 0.5))]
    )
    lock = filelock.FileLock(constants.LOCK_FILE)
    with lock:
        inference_data = tv.datasets.CIFAR10(
            root=constants.DATA_DIRECTORY, train=False, download=True, transform=transform
        )

    experimental.torch_batch_process(
        MyProcessor,
        inference_data,
        batch_size=200,
        checkpoint_interval=5,
    )


if __name__ == "__main__":
    main()
