# In this stage we add logic to save and load checkpoints in Determined.
# We also introduce preemption support for enabling pause/resume
# functionality in the WebUI.

from __future__ import print_function

import argparse
import pathlib

import torch
import torch.nn as nn
import torch.nn.functional as F
import torch.optim as optim
from torch.optim.lr_scheduler import StepLR
from torchvision import datasets, transforms

import determined as det


class Net(nn.Module):
    def __init__(self):
        super(Net, self).__init__()
        self.conv1 = nn.Conv2d(1, 32, 3, 1)
        self.conv2 = nn.Conv2d(32, 64, 3, 1)
        self.dropout1 = nn.Dropout(0.25)
        self.dropout2 = nn.Dropout(0.5)
        self.fc1 = nn.Linear(9216, 128)
        self.fc2 = nn.Linear(128, 10)

    def forward(self, x):
        x = self.conv1(x)
        x = F.relu(x)
        x = self.conv2(x)
        x = F.relu(x)
        x = F.max_pool2d(x, 2)
        x = self.dropout1(x)
        x = torch.flatten(x, 1)
        x = self.fc1(x)
        x = F.relu(x)
        x = self.dropout2(x)
        x = self.fc2(x)
        output = F.log_softmax(x, dim=1)
        return output


def train(args, model, device, train_loader, optimizer, epoch_idx, core_context):
    model.train()
    for batch_idx, (data, target) in enumerate(train_loader):
        data, target = data.to(device), target.to(device)
        optimizer.zero_grad()
        output = model(data)
        loss = F.nll_loss(output, target)
        loss.backward()
        optimizer.step()
        if (batch_idx + 1) % args.log_interval == 0:
            print(
                "Train Epoch: {} [{}/{} ({:.0f}%)]\tLoss: {:.6f}".format(
                    epoch_idx,
                    batch_idx * len(data),
                    len(train_loader.dataset),
                    100.0 * batch_idx / len(train_loader),
                    loss.item(),
                )
            )

            core_context.train.report_training_metrics(
                steps_completed=(batch_idx + 1) + epoch_idx * len(train_loader),
                metrics={"train_loss": loss.item()},
            )

            if args.dry_run:
                break


def test(args, model, device, test_loader, epoch, core_context, steps_completed):
    model.eval()
    test_loss = 0
    correct = 0
    with torch.no_grad():
        for data, target in test_loader:
            data, target = data.to(device), target.to(device)
            output = model(data)
            test_loss += F.nll_loss(output, target, reduction="sum").item()  # sum up batch loss
            pred = output.argmax(dim=1, keepdim=True)  # get the index of the max log-probability
            correct += pred.eq(target.view_as(pred)).sum().item()

    test_loss /= len(test_loader.dataset)

    print(
        "\nTest set: Average loss: {:.4f}, Accuracy: {}/{} ({:.0f}%)\n".format(
            test_loss, correct, len(test_loader.dataset), 100.0 * correct / len(test_loader.dataset)
        )
    )

    core_context.train.report_validation_metrics(
        steps_completed=steps_completed,
        metrics={"test_loss": test_loss},
    )


# Docs snippet start: define load state to restart
# NEW: Define load_state function for restarting model training from
# existing checkpoint. Returns (.pt, int).
# Also update load_state header to take trial info object as an argument.
def load_state(checkpoint_directory, trial_id):
    checkpoint_directory = pathlib.Path(checkpoint_directory)

    with checkpoint_directory.joinpath("checkpoint.pt").open("rb") as f:
        model = torch.load(f)
    with checkpoint_directory.joinpath("state").open("r") as f:
        epochs_completed, ckpt_trial_id = [int(field) for field in f.read().split(",")]
    # Docs snippet start: compare checkpoint and current trial IDs
    # If trial ID does not match our current trial ID, we'll ignore
    # epochs completed and start training from epoch_idx = 0
    if ckpt_trial_id != trial_id:
        epochs_completed = 0
    # Docs snippet end: compare checkpoint and current trial IDs

    return model, epochs_completed
    # Docs snippet end: define load state to restart


def main(core_context):
    # Training settings
    parser = argparse.ArgumentParser(description="PyTorch MNIST Example")
    parser.add_argument(
        "--batch-size",
        type=int,
        default=64,
        metavar="N",
        help="input batch size for training (default: 64)",
    )
    parser.add_argument(
        "--test-batch-size",
        type=int,
        default=1000,
        metavar="N",
        help="input batch size for testing (default: 1000)",
    )
    parser.add_argument(
        "--epochs",
        type=int,
        default=14,
        metavar="N",
        help="number of epochs to train (default: 14)",
    )
    parser.add_argument(
        "--lr", type=float, default=1.0, metavar="LR", help="learning rate (default: 1.0)"
    )
    parser.add_argument(
        "--gamma",
        type=float,
        default=0.7,
        metavar="M",
        help="Learning rate step gamma (default: 0.7)",
    )
    parser.add_argument(
        "--no-cuda", action="store_true", default=False, help="disables CUDA training"
    )
    parser.add_argument(
        "--no-mps", action="store_true", default=True, help="disables macOS GPU training"
    )
    parser.add_argument(
        "--dry-run", action="store_true", default=False, help="quickly check a single pass"
    )
    parser.add_argument("--seed", type=int, default=1, metavar="S", help="random seed (default: 1)")
    parser.add_argument(
        "--log-interval",
        type=int,
        default=100,
        metavar="N",
        help="how many batches to wait before logging training status",
    )

    args = parser.parse_args()
    use_cuda = not args.no_cuda and torch.cuda.is_available()
    use_mps = not args.no_mps and torch.backends.mps.is_available()

    # Docs snippet start: if checkpoint assign to model state
    # NEW: If checkpoint exists, load it and assign it to model state
    # prior to resuming training.
    info = det.get_cluster_info()
    assert info is not None, "this example only runs on-cluster"
    latest_checkpoint = info.latest_checkpoint
    if latest_checkpoint is None:
        epochs_completed = 0
    else:
        with core_context.checkpoint.restore_path(latest_checkpoint) as path:
            model, epochs_completed = load_state(path, info.trial.trial_id)
    # Docs snippet end: if checkpoint assign to model state

    torch.manual_seed(args.seed)

    if use_cuda:
        device = torch.device("cuda")
    elif use_mps:
        device = torch.device("mps")
    else:
        device = torch.device("cpu")

    train_kwargs = {"batch_size": args.batch_size}
    test_kwargs = {"batch_size": args.test_batch_size}
    if use_cuda:
        cuda_kwargs = {"num_workers": 1, "pin_memory": True, "shuffle": True}
        train_kwargs.update(cuda_kwargs)
        test_kwargs.update(cuda_kwargs)

    transform = transforms.Compose(
        [transforms.ToTensor(), transforms.Normalize((0.1307,), (0.3081,))]
    )
    train_dataset = datasets.MNIST("../data", train=True, download=True, transform=transform)
    test_dataset = datasets.MNIST("../data", train=False, transform=transform)
    train_loader = torch.utils.data.DataLoader(train_dataset, **train_kwargs)
    test_loader = torch.utils.data.DataLoader(test_dataset, **test_kwargs)

    model = Net().to(device)
    optimizer = optim.Adadelta(model.parameters(), lr=args.lr)

    scheduler = StepLR(optimizer, step_size=1, gamma=args.gamma)

    # NEW: Resume training from epochs_completed. This is useful
    # in the case of pausing and resuming an experiment.
    for epoch_idx in range(epochs_completed, args.epochs):
        train(args, model, device, train_loader, optimizer, epoch_idx, core_context)
        epochs_completed = epoch_idx + 1
        steps_completed = epochs_completed * len(train_loader)
        test(
            args,
            model,
            device,
            test_loader,
            epoch_idx,
            core_context,
            steps_completed=steps_completed,
        )

        scheduler.step()

        # Docs snippet start: save checkpoint
        # NEW: Save checkpoint.
        checkpoint_metadata_dict = {"steps_completed": steps_completed}

        # NEW: Here we are saving multiple files to our checkpoint
        # directory. 1) a model state file and 2) a file includes
        # information about the training loop state.
        with core_context.checkpoint.store_path(checkpoint_metadata_dict) as (path, storage_id):
            torch.save(model.state_dict(), path / "checkpoint.pt")
            with path.joinpath("state").open("w") as f:
                f.write(f"{epochs_completed},{info.trial.trial_id}")
        # Docs snippet end: save checkpoint

        # Docs snippet start: enable preemption
        # NEW: Detect when the experiment is paused by the WebUI.
        if core_context.preempt.should_preempt():
            return
        # Docs snippet end: enable preemption


if __name__ == "__main__":
    with det.core.init() as core_context:
        main(core_context=core_context)
