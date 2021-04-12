from pathlib import Path
from typing import Any, Dict, Optional, OrderedDict, Union

import numpy as np
import torch
import torch.optim
import torch.utils.data
from determined import pytorch
from determined.pytorch import LRScheduler, PyTorchTrial, PyTorchTrialContext
from sklearn.metrics import roc_auc_score
from torchvision import datasets

from base.torchvision_dataset import TorchvisionDataset
from datasets.main import load_dataset
from networks.mnist_LeNet import MNIST_LeNet, MNIST_LeNet_Autoencoder


def patch_torchvision_mnist():
    # Switch to a faster mirror.
    new_mirror = "https://ossci-datasets.s3.amazonaws.com/mnist"
    datasets.MNIST.resources = [
        ("/".join([new_mirror, url.split("/")[-1]]), md5)
        for url, md5 in datasets.MNIST.resources
    ]


patch_torchvision_mnist()


def make_dataset(download_directory) -> TorchvisionDataset:
    return load_dataset(download_directory, 0, 1, 1, 0.0, 0.01, 0.1)


class DeepSADAutoEncoderTrial(PyTorchTrial):
    def __init__(self, context: PyTorchTrialContext):
        self.context = context

        self.model = self.context.wrap_model(MNIST_LeNet_Autoencoder())

        self.optimizer = self.context.wrap_optimizer(
            torch.optim.Adam(
                self.model.parameters(),
                lr=self.context.get_hparam("learning_rate"),
                weight_decay=self.context.get_hparam("weight_decay"),
            )
        )

        self.lr_scheduler = self.context.wrap_lr_scheduler(
            torch.optim.lr_scheduler.MultiStepLR(
                self.optimizer, milestones=(50, 100), gamma=0.1
            ),
            step_mode=LRScheduler.StepMode.STEP_EVERY_EPOCH,
        )

        self.criterion = torch.nn.MSELoss(reduction="none")
        self.download_directory = f"/tmp/data-rank{self.context.distributed.get_rank()}"
        self.dataset: Optional[TorchvisionDataset] = None

    def build_training_data_loader(self) -> pytorch.DataLoader:
        if self.dataset is None:
            self.dataset = make_dataset(self.download_directory)

        return pytorch.DataLoader(
            self.dataset.train_set,
            self.context.get_per_slot_batch_size(),
            drop_last=True,
        )

    def build_validation_data_loader(self) -> pytorch.DataLoader:
        if self.dataset is None:
            self.dataset = make_dataset(self.download_directory)

        return pytorch.DataLoader(
            self.dataset.test_set,
            self.context.get_per_slot_batch_size(),
            drop_last=False,
        )

    def train_batch(
        self, batch: pytorch.TorchData, epoch_idx: int, batch_idx: int
    ) -> Union[torch.Tensor, Dict[str, Any]]:
        inputs, _, _, _ = batch
        rec = self.model(inputs)
        rec_loss = self.criterion(rec, inputs)
        loss = torch.mean(rec_loss)
        self.context.backward(loss)
        self.context.step_optimizer(self.optimizer)

        return {"loss": loss}

    def evaluate_full_dataset(
        self, data_loader: torch.utils.data.DataLoader
    ) -> Dict[str, Any]:
        criterion = self.criterion
        idx_label_score = []
        epoch_loss = 0.0
        n_batches = 0

        for data in data_loader:
            data = self.context.to_device(data)
            inputs, labels, _, idx = data
            rec = self.model(inputs)
            rec_loss = criterion(rec, inputs)
            scores = torch.mean(rec_loss, dim=tuple(range(1, rec.dim())))
            # Save triple of (idx, label, score) in a list
            idx_label_score += list(
                zip(
                    idx.cpu().data.numpy().tolist(),
                    labels.cpu().data.numpy().tolist(),
                    scores.cpu().data.numpy().tolist(),
                )
            )

            loss = torch.mean(rec_loss)
            epoch_loss += loss.item()
            n_batches += 1

        _, labels, scores = zip(*idx_label_score)
        labels = np.array(labels)
        scores = np.array(scores)
        test_auc = roc_auc_score(labels, scores)

        # Log results
        validation_loss = epoch_loss / n_batches
        print("Test Loss: {:.6f}".format(validation_loss))
        print("Test AUC: {:.2f}%".format(100.0 * test_auc))
        print("Finished testing autoencoder.")

        return {
            "validation_loss": validation_loss,
            "test_auc": test_auc,
        }


def get_filtered_ae_checkpoint_state_dict():
    # Extract encoder model weights so they can be loaded back.
    checkpoint_path = Path(__file__).resolve().parent / "ae_state_dict.pth"
    checkpoint_data = torch.load(checkpoint_path)
    state_dict = checkpoint_data["models_state_dict"][0]
    state_dict_pure = OrderedDict()
    for k, v in state_dict.items():
        if k.startswith("encoder."):
            state_dict_pure[k.replace("encoder.", "")] = v

    return state_dict_pure


class DeepSADMainTrial(DeepSADAutoEncoderTrial):
    def __init__(self, context: PyTorchTrialContext):
        self.context = context

        self.model = self.context.wrap_model(MNIST_LeNet())
        self.model.load_state_dict(get_filtered_ae_checkpoint_state_dict())

        self.optimizer = self.context.wrap_optimizer(
            torch.optim.Adam(
                self.model.parameters(),
                lr=self.context.get_hparam("learning_rate"),
                weight_decay=self.context.get_hparam("weight_decay"),
            )
        )

        self.lr_scheduler = self.context.wrap_lr_scheduler(
            torch.optim.lr_scheduler.MultiStepLR(
                self.optimizer, milestones=(50, 100), gamma=0.1
            ),
            step_mode=LRScheduler.StepMode.STEP_EVERY_EPOCH,
        )

        self.download_directory = f"/tmp/data-rank{self.context.distributed.get_rank()}"
        self.dataset: Optional[TorchvisionDataset] = None

        self.eps = 1e-6
        self.eta = self.context.get_hparam("eta")

    def build_training_data_loader(self) -> pytorch.DataLoader:
        res = super().build_training_data_loader()
        self.c = self.init_center_c(res.get_data_loader())
        return res

    def train_batch(
        self, batch: pytorch.TorchData, epoch_idx: int, batch_idx: int
    ) -> Union[torch.Tensor, Dict[str, Any]]:
        inputs, _, semi_targets, _ = batch
        self.optimizer.zero_grad()

        outputs = self.model(inputs)
        dist = torch.sum((outputs - self.c) ** 2, dim=1)
        losses = torch.where(
            semi_targets == 0,
            dist,
            self.eta * ((dist + self.eps) ** semi_targets.float()),
        )
        loss = torch.mean(losses)
        loss.backward()
        self.optimizer.step()

        return {"loss": loss}

    def evaluate_full_dataset(
        self, data_loader: torch.utils.data.DataLoader
    ) -> Dict[str, Any]:
        epoch_loss = 0.0
        n_batches = 0
        idx_label_score = []

        for data in data_loader:
            data = self.context.to_device(data)
            inputs, labels, semi_targets, idx = data

            outputs = self.model(inputs)
            dist = torch.sum((outputs - self.c) ** 2, dim=1)
            losses = torch.where(
                semi_targets == 0,
                dist,
                self.eta * ((dist + self.eps) ** semi_targets.float()),
            )
            loss = torch.mean(losses)
            scores = dist

            # Save triples of (idx, label, score) in a list
            idx_label_score += list(
                zip(
                    idx.cpu().data.numpy().tolist(),
                    labels.cpu().data.numpy().tolist(),
                    scores.cpu().data.numpy().tolist(),
                )
            )

            epoch_loss += loss.item()
            n_batches += 1

        _, labels, scores = zip(*idx_label_score)

        labels = np.array(labels)
        scores = np.array(scores)
        test_auc = roc_auc_score(labels, scores)
        epoch_loss /= n_batches

        return {
            "validation_loss": epoch_loss,
            "test_auc": test_auc,
        }

    def init_center_c(self, train_loader: torch.utils.data.DataLoader, eps=0.1):
        """Initialize hypersphere center c as the mean from an initial forward pass on the data."""
        n_samples = 0
        c = self.context.to_device(torch.zeros(self.model.rep_dim))

        self.model.eval()

        with torch.no_grad():
            for data in train_loader:
                data = self.context.to_device(data)
                # get the inputs of the batch
                inputs, _, _, _ = data
                outputs = self.model(inputs)
                n_samples += outputs.shape[0]
                c += torch.sum(outputs, dim=0)

        c /= n_samples

        # If c_i is too close to 0, set to +-eps. Reason: a zero unit can be trivially matched with zero weights.
        c[(abs(c) < eps) & (c < 0)] = -eps
        c[(abs(c) < eps) & (c > 0)] = eps

        return c
