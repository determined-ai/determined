# ProtoNet implementation based on https://github.com/jakesnell/prototypical-networks/blob/master/protonets/models/few_shot.py

import os

import torch
import torch.nn as nn
import torch.nn.functional as F

import numpy as np
from determined.pytorch import DataLoader, LRScheduler, PyTorchTrial, PyTorchTrialContext, TorchData
from data import OmniglotTasks


class Flatten(nn.Module):
    def __init__(self):
        super(Flatten, self).__init__()

    def forward(self, x):
        return x.view(x.size(0), -1)


def SquaredDistance(x, y):
    # x: N x D
    # y: M x D
    n = x.size(0)
    m = y.size(0)
    d = x.size(1)
    assert d == y.size(1)

    x = x.unsqueeze(1).expand(n, m, d)
    y = y.unsqueeze(0).expand(n, m, d)

    return torch.pow(x - y, 2).sum(2)


class OmniglotProtoNetTrial(PyTorchTrial):
    def __init__(self, context: PyTorchTrialContext) -> None:
        self.context = context
        self.data_config = context.get_data_config()
        self.num_classes = {
            "train": context.get_hparam("num_classes_train"),
            "val": context.get_hparam("num_classes_val"),
        }
        self.num_support = {
            "train": context.get_hparam("num_support_train"),
            "val": context.get_hparam("num_support_val"),
        }
        self.num_query = {
            "train": context.get_hparam("num_query_train"),
            "val": None,  # Use all available examples for val at meta-test time
        }
        self.get_train_valid_splits()

        x_dim = 1  # Omniglot is black and white
        hid_dim = self.context.get_hparam("hidden_dim")
        z_dim = self.context.get_hparam("embedding_dim")

        def conv_block(in_channels, out_channels):
            return nn.Sequential(
                nn.Conv2d(in_channels, out_channels, 3, padding=1),
                nn.BatchNorm2d(out_channels),
                nn.ReLU(),
                nn.MaxPool2d(2),
            )

        self.model = self.context.Model(nn.Sequential(
            conv_block(x_dim, hid_dim),
            conv_block(hid_dim, hid_dim),
            conv_block(hid_dim, hid_dim),
            conv_block(hid_dim, z_dim),
            Flatten(),
        ))

        self.optimizer = self.context.Optimizer(torch.optim.Adam(
            self.model.parameters(),
            lr=self.context.get_hparam("learning_rate"),
            weight_decay=self.context.get_hparam("weight_decay"),
        ))

        self.lr_scheduler = self.context.LRScheduler(
            torch.optim.lr_scheduler.StepLR(
                self.optimizer,
                self.context.get_hparam("reduce_every"),
                gamma=self.context.get_hparam("lr_gamma"),
            ),
            LRScheduler.StepMode.STEP_EVERY_EPOCH
        )

    def get_train_valid_splits(self):
        n_classes = 0
        for root, dirs, files in os.walk(self.data_config["data_path"]):
            if len(dirs) == 0:
                n_classes += 1
        idxs = np.arange(n_classes)
        print("num classes in dataset: {}".format(n_classes))
        np.random.shuffle(idxs)
        n_val_classes = int(n_classes * self.data_config["validation_portion"])
        self.val_class_idxs = idxs[0:n_val_classes]
        self.train_class_idxs = idxs[n_val_classes:]

    def build_training_data_loader(self) -> DataLoader:
        dataset = OmniglotTasks(
            self.data_config["data_path"],
            self.data_config["tasks_per_epoch_train"],
            self.train_class_idxs,
            self.context.get_hparam("img_resize_dim"),
            self.num_classes["train"],
            self.num_support["train"],
            self.num_query["train"],
        )
        return DataLoader(
            dataset,
            self.context.get_per_slot_batch_size(),
            num_workers=self.data_config["train_workers"],
            collate_fn=dataset.get_collate_fn(),
        )

    def build_validation_data_loader(self) -> DataLoader:
        dataset = OmniglotTasks(
            self.data_config["data_path"],
            self.data_config["tasks_per_epoch_val"],
            self.val_class_idxs,
            self.context.get_hparam("img_resize_dim"),
            self.num_classes["val"],
            self.num_support["val"],
            self.num_query["val"],
        )
        return DataLoader(
            dataset,
            self.context.get_hparam("val_batch_size"),
            num_workers=self.data_config["val_workers"],
            collate_fn=dataset.get_collate_fn(),
        )

    def loss(self, x_support, y_support, x_query, y_query, model, split):
        # x dimension N x C x H x W
        _, channels, height, width = x_support.size()

        num_classes = self.num_classes[split]
        num_support = int(y_support.size(0) / num_classes)
        num_query = int(y_query.size(0) / num_classes)

        # First resort x so examples are ordered by class.
        support_idxs = torch.argsort(y_support)
        query_idxs = torch.argsort(y_query)
        x_support = x_support[support_idxs]
        y_support = y_support[support_idxs]
        x_query = x_query[query_idxs]
        y_query = y_query[query_idxs]

        # Group support and query data into one forward pass

        x = torch.cat([x_support, x_query], 0)

        embedding = model(x)
        embedding_dim = embedding.size(-1)

        # Now we can reshape to get prototype embeddings
        # Prototype size: (num_classes, embedding_dim)
        prototypes = (
            embedding[0 : num_classes * num_support]
            .view(num_classes, num_support, embedding_dim)
            .mean(1)
        )

        # Embedded query size: (num_classes * num_query, embedding_dim)
        embedded_query = embedding[num_classes * num_support :]

        # Compute distance between query set and prototypes
        # Distance size: (num_classes * num_query, num_classes)
        euclidean_dist = SquaredDistance(embedded_query, prototypes)

        # Class log probabilities by treating -distances as logits
        # Log_prob_query size: (num_classes, num_query, num_classes)
        log_prob_query = F.log_softmax(-euclidean_dist, dim=1).view(
            num_classes, num_query, -1
        )

        # Match query examples with classes
        y_query_expand = y_query.view(num_classes, num_query, 1)
        loss = -log_prob_query.gather(2, y_query_expand).squeeze().view(-1).mean()

        _, pred_query = log_prob_query.max(2)

        acc = torch.eq(pred_query, y_query_expand.squeeze()).float().mean()

        return loss, acc

    def train_batch(
        self, batch: TorchData, model: nn.Module, epoch_idx: int, batch_idx: int
    ):
        # Typically ProtoNet is run with batch_size = 1
        total_loss = 0
        total_acc = 0
        n_tasks = len(batch)

        for t in range(n_tasks):
            loss, acc = self.loss(
                batch[t]["support"][0],
                batch[t]["support"][1],
                batch[t]["query"][0],
                batch[t]["query"][1],
                self.model,
                "train",
            )
            total_loss += loss
            total_acc += acc

        outputs = {"loss": total_loss / n_tasks, "acc": total_acc / n_tasks}
        self.context.backward(outputs["loss"])
        self.context.step_optimizer(self.optimizer)
        return outputs

    def evaluate_full_dataset(
        self,
        data_loader: torch.utils.data.dataloader.DataLoader,
        model: torch.nn.modules.module.Module,
    ):
        total_loss = 0
        total_acc = 0

        for batch in data_loader:
            n_tasks = len(batch)
            for t in range(n_tasks):
                # Need to pass to GPU because we are getting a pytorch dataloader
                # instead of our own TorchData object like in train_batch.
                loss, acc = self.loss(
                    batch[t]["support"][0].cuda(),
                    batch[t]["support"][1].cuda(),
                    batch[t]["query"][0].cuda(),
                    batch[t]["query"][1].cuda(),
                    self.model,
                    "val",
                )
                total_loss += loss
                total_acc += acc
        return {
            "loss": total_loss / (len(data_loader) * n_tasks),
            "acc": total_acc / (len(data_loader) * n_tasks),
        }
