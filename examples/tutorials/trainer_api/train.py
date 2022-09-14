import os

from determined import core, pytorch
import torch
from typing import Any, Dict, Tuple, cast
from torchvision import datasets, transforms
from torch import nn
from determined.pytorch import DataLoader, TorchData, PyTorchTrialContext
import determined as det
import torch.distributed as dist
import logging


class Flatten(nn.Module):
    def forward(self, *args: TorchData, **kwargs: Any) -> torch.Tensor:
        assert len(args) == 1
        x = args[0]
        assert isinstance(x, torch.Tensor)
        return x.contiguous().view(x.size(0), -1)


class MNistTrial(pytorch.PyTorchTrial):
    def __init__(self, context: PyTorchTrialContext) -> None:
        self.context = context
        self.download_directory = f"/tmp/mnist-data"
        os.makedirs(self.download_directory, exist_ok=True)

        self.model = context.wrap_model(nn.Sequential(
            nn.Conv2d(1, 32, 3, 1),
            nn.ReLU(),
            nn.Conv2d(
                32, 64, 3, 1
            ),
            nn.ReLU(),
            nn.MaxPool2d(2),
            nn.Dropout2d(0.25),
            Flatten(),
            nn.Linear(9216, 128),
            nn.ReLU(),
            nn.Dropout2d(0.5),
            nn.Linear(128, 10),
            nn.LogSoftmax(),
        ))

        self.optimizer = context.wrap_optimizer(torch.optim.Adadelta(
            self.model.parameters(), lr=0.1)
        )
        self.batch_size = 100

    def train_batch(
            self, batch: pytorch.TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        batch = cast(Tuple[torch.Tensor, torch.Tensor], batch)
        data, labels = batch

        data = self.context.to_device(data)
        labels = self.context.to_device(labels)

        output = self.model(data)
        loss = torch.nn.functional.nll_loss(output, labels)

        self.context.backward(loss)
        self.context.step_optimizer(self.optimizer)

        return {"loss": loss}

    def evaluate_batch(self, batch: pytorch.TorchData) -> Dict[str, Any]:
        batch = cast(Tuple[torch.Tensor, torch.Tensor], batch)
        data, labels = batch

        data = self.context.to_device(data)
        labels = self.context.to_device(labels)

        output = self.model(data)
        validation_loss = torch.nn.functional.nll_loss(output, labels).item()

        pred = output.argmax(dim=1, keepdim=True)
        accuracy = pred.eq(labels.view_as(pred)).sum().item() / len(data)

        return {"validation_loss": validation_loss, "accuracy": accuracy}

    def build_training_data_loader(self) -> DataLoader:
        import filelock
        with filelock.FileLock(os.path.join(self.download_directory, "lock")):
            train_set = datasets.MNIST(
                self.download_directory,
                train=True,
                download=True,
                transform=transforms.Compose(
                    [
                        transforms.ToTensor(),
                        # These are the precomputed mean and standard deviation of the
                        # MNIST data; this normalizes the data to have zero mean and unit
                        # standard deviation.
                        transforms.Normalize((0.1307,), (0.3081,)),
                    ]
                ),
            )
        return DataLoader(train_set, batch_size=self.batch_size)

    def build_validation_data_loader(self) -> DataLoader:
        import filelock
        with filelock.FileLock(os.path.join(self.download_directory, "lock")):
            validation_set = datasets.MNIST(
                self.download_directory,
                train=False,
                download=True,
                transform=transforms.Compose(
                    [
                        transforms.ToTensor(),
                        # These are the precomputed mean and standard deviation of the
                        # MNIST data; this normalizes the data to have zero mean and unit
                        # standard deviation.
                        transforms.Normalize((0.1307,), (0.3081,)),
                    ]
                )
            )
        return DataLoader(validation_set, batch_size=self.batch_size)


def main():
    dist.init_process_group(backend="gloo")
    import os
    os.environ["USE_TORCH_DISTRIBUTED"] = "true"
    with det.pytorch.init(hparams={"global_batch_size": 32},
                          distributed=core.DistributedContext.from_torch_distributed(chief_ip="localhost")) as train_context:
        trial = MNistTrial(train_context)
        trainer = det.pytorch.Trainer(trial, train_context)
        # trainer.configure_profiler(enabled=True,
        #                            sync_timings=True,
        #                            begin_on_batch=0,
        #                            end_after_batch=10)
        trainer.fit(
            max_length=pytorch.Epoch(1),
            checkpoint_period=pytorch.Batch(10),
            validation_period=pytorch.Batch(10),
        )


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)
    main()
