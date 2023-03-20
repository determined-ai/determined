import torch
from torch import nn

from determined import pytorch


class OnesDataset(torch.utils.data.Dataset):
    def __len__(self):
        return 64

    def __getitem__(self, index):
        return torch.Tensor([1.0])


class OneVarPytorchTrial(pytorch.PyTorchTrial):
    def __init__(self, context):
        self.context = context

        self.model = context.wrap_model(nn.Linear(1, 1, False))
        self.opt = context.wrap_optimizer(
            torch.optim.SGD(self.model.parameters(), lr=0.001), backward_passes_per_step=2
        )

    def train_batch(self, batch, epoch_idx, batch_idx):
        loss = torch.nn.MSELoss()(self.model(batch), batch)
        self.context.step_optimizer(self.opt)
        return {"loss": loss}

    def evaluate_batch(self, batch):
        data = labels = batch
        loss = torch.nn.MSELoss()(self.model(data), labels)
        return {"loss": loss}

    def build_training_data_loader(self):
        return pytorch.DataLoader(OnesDataset(), batch_size=self.context.get_per_slot_batch_size())

    def build_validation_data_loader(self):
        return pytorch.DataLoader(OnesDataset(), batch_size=self.context.get_per_slot_batch_size())
