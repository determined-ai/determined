# In this stage we introduce distributed training. You should be able
# to see multiple slots active in the Cluster tab corresponding to
# the value you set for slots_per_trial you set in distributed.yaml,
# as well as logs appearing from multiple ranks, in the WebUI.

from __future__ import print_function

import argparse
import os
import pathlib

import filelock

# Docs snippet start: import torch distrib
# NEW: Import torch distributed libraries.
import torch
import torch.distributed as dist
import torch.nn as nn
import torch.nn.functional as F
import torch.optim as optim
from torch.nn.parallel import DistributedDataParallel as DDP
from torch.optim.lr_scheduler import StepLR
from torch.utils.data.distributed import DistributedSampler
from torchvision import datasets, transforms

import determined as det

# Docs snippet end: import torch distrib


class Net(nn.Module):
    def __init__(self, hparams):
        super(Net, self).__init__()
        self.conv1 = nn.Conv2d(1, hparams["n_filters1"], 3, 1)
        self.conv2 = nn.Conv2d(hparams["n_filters1"], hparams["n_filters2"], 3, 1)
        self.dropout1 = nn.Dropout(hparams["dropout1"])
        self.dropout2 = nn.Dropout(hparams["dropout2"])
        self.fc1 = nn.Linear(144 * hparams["n_filters2"], 128)
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


def train(args, model, device, train_loader, optimizer, core_context, epoch_idx, op):
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
            # Docs snippet start: report metrics rank 0
            # NEW: Report metrics only on rank 0: only the chief worker
            # may report training metrics and progress, or upload checkpoints.
            if core_context.distributed.rank == 0:
                core_context.train.report_training_metrics(
                    steps_completed=(batch_idx + 1) + epoch_idx * len(train_loader),
                    metrics={"train_loss": loss.item()},
                )
            # Docs snippet end: report metrics rank 0

            if args.dry_run:
                break

    # NEW: Report progress only on rank 0.
    if core_context.distributed.rank == 0:
        op.report_progress(epoch_idx)


def test(args, model, device, test_loader, core_context, steps_completed, op) -> int:
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

    # NEW: Report metrics only on rank 0.
    if core_context.distributed.rank == 0:
        core_context.train.report_validation_metrics(
            steps_completed=steps_completed, metrics={"test_loss": test_loss}
        )

    return test_loss


def load_state(checkpoint_directory, trial_id):
    checkpoint_directory = pathlib.Path(checkpoint_directory)

    with checkpoint_directory.joinpath("checkpoint.pt").open("rb") as f:
        model = torch.load(f)
    with checkpoint_directory.joinpath("state").open("r") as f:
        epochs_completed, ckpt_trial_id = [int(field) for field in f.read().split(",")]

    # If trial ID does not match our current trial ID, we'll ignore epochs
    # completed and start training from epoch_idx = 0
    if ckpt_trial_id != trial_id:
        epochs_completed = 0

    return model, epochs_completed


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

    info = det.get_cluster_info()
    assert info is not None, "this example only runs on-cluster"
    latest_checkpoint = info.latest_checkpoint
    if latest_checkpoint is None:
        epochs_completed = 0
    else:
        with core_context.checkpoint.restore_path(latest_checkpoint) as path:
            model, epochs_completed = load_state(path, info.trial.trial_id)

    torch.manual_seed(args.seed)

    if use_cuda:
        # Docs snippet start: set device
        # NEW: Change selected device to the one with index of local_rank.
        device = torch.device(core_context.distributed.local_rank)
    elif use_mps:
        device = torch.device("mps")
    else:
        device = torch.device("cpu")
        # Docs snippet end: set device

    train_kwargs = {"batch_size": args.batch_size}
    test_kwargs = {"batch_size": args.test_batch_size}
    if use_cuda:
        # NEW: Remove DataLoader shuffle argument since it is mutually
        # exlusive with DistributedSampler shuffle, set shuffle=True
        # there instead.
        cuda_kwargs = {"num_workers": 1, "pin_memory": True}
        train_kwargs.update(cuda_kwargs)
        test_kwargs.update(cuda_kwargs)

    transform = transforms.Compose(
        [transforms.ToTensor(), transforms.Normalize((0.1307,), (0.3081,))]
    )

    with filelock.FileLock(os.path.join(os.getcwd(), "lock")):
        train_dataset = datasets.MNIST("../data", train=True, download=True, transform=transform)
        test_dataset = datasets.MNIST("../data", train=False, transform=transform)

    # Docs snippet start: shard data
    # NEW: Create DistributedSampler object for sharding data into
    # core_context.distributed.size parts.
    train_sampler = DistributedSampler(
        train_dataset,
        num_replicas=core_context.distributed.size,
        rank=core_context.distributed.rank,
        shuffle=True,
    )
    test_sampler = DistributedSampler(
        test_dataset,
        num_replicas=core_context.distributed.size,
        rank=core_context.distributed.rank,
        shuffle=True,
    )

    # NEW: Shard data.
    train_loader = torch.utils.data.DataLoader(train_dataset, sampler=train_sampler, **train_kwargs)
    test_loader = torch.utils.data.DataLoader(test_dataset, sampler=test_sampler, **test_kwargs)
    # Docs snippet end: shard data

    hparams = info.trial.hparams

    # Docs snippet start: DDP
    model = Net(hparams).to(device)
    # NEW: Wrap model with DDP. Aggregates gradients and synchronizes
    # model training across slots.
    model = DDP(model, device_ids=[device], output_device=device)
    # Docs snippet end: DDP

    optimizer = optim.Adadelta(model.parameters(), lr=hparams["learning_rate"])
    scheduler = StepLR(optimizer, step_size=1, gamma=args.gamma)

    epoch_idx = epochs_completed
    last_checkpoint_batch = None

    for op in core_context.searcher.operations():
        while epoch_idx < op.length:
            train(args, model, device, train_loader, optimizer, core_context, epoch_idx, op)
            epochs_completed = epoch_idx + 1
            steps_completed = epochs_completed * len(train_loader)
            test_loss = test(args, model, device, test_loader, core_context, steps_completed, op)

            scheduler.step()

            checkpoint_metadata_dict = {
                "steps_completed": steps_completed,
            }

            epoch_idx += 1

            # Store checkpoints only on rank 0.
            if core_context.distributed.rank == 0:
                with core_context.checkpoint.store_path(checkpoint_metadata_dict) as (
                    path,
                    storage_id,
                ):
                    torch.save(model.state_dict(), path / "checkpoint.pt")
                    with path.joinpath("state").open("w") as f:
                        f.write(f"{epochs_completed},{info.trial.trial_id}")

            if core_context.preempt.should_preempt():
                return

        # Report completed only on rank 0.
        if core_context.distributed.rank == 0:
            op.report_completed(test_loss)


# Docs snippet start: initialize process group
if __name__ == "__main__":
    # NEW: Initialize process group using torch.
    dist.init_process_group("nccl")

    # NEW: Initialize distributed context using from_torch_distributed
    # (obtains info such as rank, size, etc. from default torch
    # environment variables).
    distributed = det.core.DistributedContext.from_torch_distributed()
    with det.core.init(distributed=distributed) as core_context:
        main(core_context)
    # Docs snippet end: initialize process group
