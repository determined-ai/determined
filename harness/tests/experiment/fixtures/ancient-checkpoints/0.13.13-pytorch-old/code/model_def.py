import torch
from torch import nn

from determined.pytorch import DataLoader, PyTorchTrial


class IndexDataset(torch.utils.data.Dataset):
    def __len__(self):
        return 64

    def __getitem__(self, index):
        return torch.Tensor([float(1)])


class OneVarPytorchTrial(PyTorchTrial):
    def __init__(self, context):
        self.context = context

    def build_model(self):
        model = nn.Linear(1, 1, False)
        # initialize weights to 0
        model.weight.data.fill_(0)
        print("weight starts at:", model.weight.data[0])
        return model

    def optimizer(self, model):
        return torch.optim.SGD(model.parameters(), 0.001)

    def train_batch(self, batch, model, epoch_idx, batch_idx):
        # Figure what the weight should be right now
        w_real = model.weight.data[0]

        loss = torch.nn.L1Loss()(model(batch), batch)

        return {"loss": loss}

    def evaluate_batch(self, batch, model):
        # Return something... anything.
        val = batch[0]
        return {"loss": val}

    def build_training_data_loader(self):
        return DataLoader(IndexDataset(), batch_size=self.context.get_per_slot_batch_size())

    def build_validation_data_loader(self):
        return DataLoader(IndexDataset(), batch_size=self.context.get_per_slot_batch_size())
