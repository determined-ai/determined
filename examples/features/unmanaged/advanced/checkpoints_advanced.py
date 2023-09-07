#!/usr/bin/env python3

from typing import Tuple

import torch

from determined.experimental import core_v2


class IdentityDataset(torch.utils.data.Dataset):
    def __init__(self, initial_value: int = 1):
        self.initial_value = initial_value

    def __len__(self) -> int:
        return 64

    def __getitem__(self, index: int) -> Tuple:
        v = float(self.initial_value + 0.1 * index)
        return torch.Tensor([v]), torch.Tensor([v])


def main():
    core_v2.init(
        defaults=core_v2.DefaultConfig(
            name="unmanaged-checkpoints-advanced",
            hparams={
                "lr": 1e-5,
                "max_epochs": 10,
            },
            labels=["some", "set", "of", "labels"],
            description="torch identity example",
        ),
        unmanaged=core_v2.UnmanagedConfig(
            external_experiment_id="unmanaged-checkpoints-advanced",
            external_trial_id="unmanaged-checkpoints-advanced",
        ),
    )

    lr = core_v2.info.trial.hparams["lr"]
    max_epochs = core_v2.info.trial.hparams["max_epochs"]

    model = torch.nn.Linear(1, 1, False)

    # Load / setup initial state.
    latest_checkpoint = core_v2.info.latest_checkpoint
    initial_epoch = 0
    if latest_checkpoint is not None:
        initial_epoch = core_v2.checkpoint.get_metadata(latest_checkpoint)["epoch_idx"] + 1
        print(f"Continuing from epoch_idx {initial_epoch}")
        with core_v2.checkpoint.restore_path(latest_checkpoint) as path:
            model.load_state_dict(torch.load(path / "checkpoint.pt"))
    else:
        initial_epoch = 0
        model.weight.data.fill_(0)

    optimizer = torch.optim.SGD(model.parameters(), lr)
    loss_fn = torch.nn.MSELoss(reduction="mean")
    train_dataset, val_dataset = IdentityDataset(), IdentityDataset(42)
    train_dataloader = torch.utils.data.DataLoader(train_dataset, batch_size=8, shuffle=True)
    val_dataloader = torch.utils.data.DataLoader(val_dataset, batch_size=8, shuffle=True)

    model.train()

    for epoch_idx in range(initial_epoch, max_epochs):
        print(f"Starting epoch: {epoch_idx}")
        for batch_idx, (data, label) in enumerate(train_dataloader):
            loss = loss_fn(model(data), label)
            loss.backward()
            optimizer.step()
            optimizer.zero_grad()

            steps_completed = (batch_idx + 1) + epoch_idx * len(train_dataloader)
            core_v2.train.report_training_metrics(
                steps_completed=steps_completed,
                metrics={"loss": loss.item(), "weight": model.weight.data.item()},
            )

        steps_completed = epoch_idx * len(train_dataloader)
        model.eval()
        with torch.no_grad():
            val_loss = 0
            for batch_idx, (data, label) in enumerate(val_dataloader):
                loss = loss_fn(model(data), label)
                val_loss += loss.item()

            core_v2.train.report_validation_metrics(
                steps_completed=steps_completed,
                metrics={"loss": val_loss, "weight": model.weight.data.item()},
            )
            print(f"Done epoch: {epoch_idx}, val loss: {val_loss}")
        with core_v2.checkpoint.store_path(
            dict(epoch_idx=epoch_idx, steps_completed=steps_completed)
        ) as (path, _):
            torch.save(model.state_dict(), path / "checkpoint.pt")

    core_v2.close()


if __name__ == "__main__":
    main()
