import tempfile
from math import ceil

import torch
import torch.nn.functional as F
from determined.pytorch import DataLoader, PyTorchTrial, PyTorchTrialContext
from torch.nn import Linear
from torch.utils.data import random_split
from torch_geometric.data.dataloader import Collater
from torch_geometric.datasets import TUDataset
from torch_geometric.nn import DenseGraphConv, GCNConv, dense_mincut_pool
from torch_geometric.utils import to_dense_adj, to_dense_batch


# Ported from https://github.com/rusty1s/pytorch_geometric/blob/master/examples/proteins_mincut_pool.py
class Net(torch.nn.Module):
    def __init__(self, in_channels, out_channels, hidden_channels, average_nodes):
        super(Net, self).__init__()

        self.conv1 = GCNConv(in_channels, hidden_channels)
        num_nodes = ceil(0.5 * average_nodes)
        self.pool1 = Linear(hidden_channels, num_nodes)

        self.conv2 = DenseGraphConv(hidden_channels, hidden_channels)
        num_nodes = ceil(0.5 * num_nodes)
        self.pool2 = Linear(hidden_channels, num_nodes)

        self.conv3 = DenseGraphConv(hidden_channels, hidden_channels)

        self.lin1 = Linear(hidden_channels, hidden_channels)
        self.lin2 = Linear(hidden_channels, out_channels)

    def forward(self, x, edge_index, batch):
        x = F.relu(self.conv1(x, edge_index))

        x, mask = to_dense_batch(x, batch)
        adj = to_dense_adj(edge_index, batch)

        s = self.pool1(x)
        x, adj, mc1, o1 = dense_mincut_pool(x, adj, s, mask)

        x = F.relu(self.conv2(x, adj))
        s = self.pool2(x)

        x, adj, mc2, o2 = dense_mincut_pool(x, adj, s)

        x = self.conv3(x, adj)

        x = x.mean(dim=1)
        x = F.relu(self.lin1(x))
        x = self.lin2(x)
        return F.log_softmax(x, dim=-1), mc1 + mc2, o1 + o2


class GCNTrial(PyTorchTrial):
    def __init__(self, context: PyTorchTrialContext):
        self.context = context

        # Create a unique download directory for each rank so they don't overwrite each
        # other when doing distributed training.
        download_directory = tempfile.mkdtemp()

        self.dataset = TUDataset(
            root=download_directory,
            name=self.context.get_hparam("dataset"),
            use_node_attr=True,
        )

        num_training = self.context.get_hparam("training_records")
        num_val = len(self.dataset) - num_training
        self.train_subset, self.valid_subset = random_split(
            self.dataset, [num_training, num_val]
        )

        self.num_feature = self.dataset.num_features
        self.num_class = self.dataset.num_classes
        average_nodes = int(self.dataset.data.x.size(0) / len(self.dataset))

        self.model = self.context.wrap_model(
            Net(
                self.num_feature,
                self.num_class,
                self.context.get_hparam("hidden_channels"),
                average_nodes,
            )
        )

        self.optimizer = self.context.wrap_optimizer(
            torch.optim.Adam(
                self.model.parameters(),
                lr=self.context.get_hparam("lr"),
                weight_decay=self.context.get_hparam("weight_decay"),
            )
        )

    def train_batch(self, batch, epoch_idx: int, batch_idx: int):
        # NB: `batch` is `torch_geometric.data.batch.Batch` type
        out, mc_loss, o_loss = self.model(batch.x, batch.edge_index, batch.batch)
        nll_loss = F.nll_loss(out, batch.y.view(-1))
        loss = nll_loss + mc_loss + o_loss

        self.context.backward(loss)
        self.context.step_optimizer(self.optimizer)

        pred = out.max(dim=1)[1]
        correct = pred.eq(batch.y).sum().item() / len(batch.y)
        return {
            "accuracy": correct,
            "nll_loss": nll_loss,
            "mc_loss": mc_loss,
            "o_loss": o_loss,
            "loss": loss,
            "batch_num_graphs": batch.num_graphs,
        }

    def evaluate_batch(self, batch):
        # NB: `batch` is `torch_geometric.data.batch.Batch` type
        out, mc_loss, o_loss = self.model(batch.x, batch.edge_index, batch.batch)
        loss = F.nll_loss(out, batch.y.view(-1)) + mc_loss + o_loss

        pred = out.max(dim=1)[1]
        correct = pred.eq(batch.y).sum().item() / len(batch.y)
        return {
            "accuracy": correct,
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
