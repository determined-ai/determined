import tempfile

import torch
import torch.nn.functional as F
from torch.utils.data import random_split
from torch_geometric.datasets import TUDataset
from torch_geometric.loader.dataloader import Collater
from torch_geometric.nn import GraphConv, TopKPooling
from torch_geometric.nn import global_max_pool as gmp
from torch_geometric.nn import global_mean_pool as gap

from determined.pytorch import DataLoader, PyTorchTrial, PyTorchTrialContext


# Ported from https://github.com/rusty1s/pytorch_geometric/blob/master/examples/proteins_topk_pool.py
class Net(torch.nn.Module):
    def __init__(self, in_channels, out_channels, topk_pooling_ratio=0.8, dropout=0.5):
        super(Net, self).__init__()
        self.dropout = dropout

        self.conv1 = GraphConv(in_channels, 128)
        self.pool1 = TopKPooling(128, ratio=topk_pooling_ratio)
        self.conv2 = GraphConv(128, 128)
        self.pool2 = TopKPooling(128, ratio=topk_pooling_ratio)
        self.conv3 = GraphConv(128, 128)
        self.pool3 = TopKPooling(128, ratio=topk_pooling_ratio)

        self.lin1 = torch.nn.Linear(256, 128)
        self.lin2 = torch.nn.Linear(128, 64)
        self.lin3 = torch.nn.Linear(64, out_channels)

    def forward(self, data):
        x, edge_index, batch = data.x, data.edge_index, data.batch

        x = F.relu(self.conv1(x, edge_index))
        x, edge_index, _, batch, _, _ = self.pool1(x, edge_index, None, batch)
        x1 = torch.cat([gmp(x, batch), gap(x, batch)], dim=1)

        x = F.relu(self.conv2(x, edge_index))
        x, edge_index, _, batch, _, _ = self.pool2(x, edge_index, None, batch)
        x2 = torch.cat([gmp(x, batch), gap(x, batch)], dim=1)

        x = F.relu(self.conv3(x, edge_index))
        x, edge_index, _, batch, _, _ = self.pool3(x, edge_index, None, batch)
        x3 = torch.cat([gmp(x, batch), gap(x, batch)], dim=1)

        x = x1 + x2 + x3

        x = F.relu(self.lin1(x))
        x = F.dropout(x, p=self.dropout, training=self.training)
        x = F.relu(self.lin2(x))
        x = F.log_softmax(self.lin3(x), dim=-1)

        return x


def download_data_with_retry(n_retries, download_directory, dataset_name):
    while n_retries > 0:
        try:
            return TUDataset(root=download_directory, name=dataset_name)
        except Exception as e:
            n_retries -= 1
            if n_retries == 0:
                raise


class GraphConvTrial(PyTorchTrial):
    def __init__(self, context: PyTorchTrialContext):
        self.context = context

        # Create a unique download directory for each rank so they don't overwrite each
        # other when doing distributed training.
        download_directory = tempfile.mkdtemp()

        self.dataset = download_data_with_retry(
            3,
            download_directory,
            self.context.get_hparam("dataset"),
        )

        num_training = self.context.get_hparam("training_records")
        num_val = len(self.dataset) - num_training
        self.train_subset, self.valid_subset = random_split(self.dataset, [num_training, num_val])

        self.num_feature = self.dataset.num_features
        self.num_class = self.dataset.num_classes

        self.model = self.context.wrap_model(
            Net(
                self.dataset.num_features,
                self.dataset.num_classes,
                self.context.get_hparam("topk_pooling_ratio"),
                self.context.get_hparam("dropout"),
            )
        )

        self.optimizer = self.context.wrap_optimizer(
            torch.optim.Adam(
                self.model.parameters(),
                lr=self.context.get_hparam("lr"),
            )
        )

    def train_batch(self, batch, epoch_idx: int, batch_idx: int):
        # NB: `batch` is `torch_geometric.data.batch.Batch` type
        output = self.model(batch)
        loss = F.nll_loss(output, batch.y)

        self.context.backward(loss)
        self.context.step_optimizer(self.optimizer)
        return {
            "loss": loss,
        }

    def evaluate_batch(self, batch):
        # NB: `batch` is `torch_geometric.data.batch.Batch` type
        output = self.model(batch)
        loss = F.nll_loss(output, batch.y)

        pred = output.max(dim=1)[1]
        accuracy = pred.eq(batch.y).sum().item() / len(batch.y)
        return {
            "accuracy": accuracy,
            "validation_loss": loss,
        }

    def build_training_data_loader(self):
        return DataLoader(
            self.train_subset,
            batch_size=self.context.get_per_slot_batch_size(),
            collate_fn=Collater([], []),
        )

    def build_validation_data_loader(self):
        return DataLoader(
            self.valid_subset,
            batch_size=self.context.get_per_slot_batch_size(),
            collate_fn=Collater([], []),
        )

    def get_batch_length(self, batch):
        # Since `torch_geometric.data.batch.Batch` has a custom way of exposing
        # the batch dimension size, the users must override this method,
        # so the trial could properly calculate the batch sizes at runtime.

        return batch.num_graphs
