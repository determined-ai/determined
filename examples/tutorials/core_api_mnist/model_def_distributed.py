# Introduce distributed training

from __future__ import print_function
import argparse
import torch
import torch.nn as nn
import torch.nn.functional as F
import torch.optim as optim
from torchvision import datasets, transforms
from torch.optim.lr_scheduler import StepLR
import determined as det
import pathlib

# NEW: Import torch distributed libraries and os
import torch.distributed as dist
from torch.utils.data.distributed import DistributedSampler
from torch.nn.parallel import DistributedDataParallel as DDP
import os


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


def train(args, model, device, train_loader, optimizer, core_context, epoch, op):
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
                    epoch,
                    (batch_idx) * len(data),
                    len(train_loader.dataset),
                    100.0 * (batch_idx) / len(train_loader),
                    loss.item(),
                )
            )

            # NEW: Report metrics only on rank 0: only the chief worker may report training metrics and progress,
            # or upload checkpoints.
            if core_context.distributed.rank == 0:
                core_context.train.report_training_metrics(
                    steps_completed=(batch_idx + 1) + (epoch - 1) * len(train_loader),
                    metrics={"train_loss": loss.item()},
                )

            # NEW: Report progress only on rank 0
            if core_context.distributed.rank == 0:
                op.report_progress(epoch)

            if args.dry_run:
                break


def test(args, model, device, test_loader, core_context, steps_completed, op) -> int:

    model.eval()
    test_loss = 0
    correct = 0
    with torch.no_grad():
        for _, (data, target) in enumerate(test_loader):
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

    # NEW: report metrics only on rank 0
    if core_context.distributed.rank == 0:
        core_context.train.report_validation_metrics(
            steps_completed=steps_completed, metrics={"test_loss": test_loss}
        )

    return test_loss


def load_state(checkpoint_directory):
    checkpoint_directory = pathlib.Path(checkpoint_directory)
    with checkpoint_directory.joinpath("state").open("r") as f:
        return torch.load(f)


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
    parser.add_argument(
        "--save-model", action="store_true", default=True, help="For Saving the current Model"
    )

    args = parser.parse_args()
    use_cuda = not args.no_cuda and torch.cuda.is_available()
    use_mps = not args.no_mps and torch.backends.mps.is_available()

    info = det.get_cluster_info()
    assert info is not None, "this example only runs on-cluster"
    latest_checkpoint = info.latest_checkpoint
    if latest_checkpoint is not None:
        with core_context.checkpoint.restore_path(latest_checkpoint) as path:
            model = load_state(path)

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
        # NEW: sampler is mututually exclusive with shuffle
        cuda_kwargs = {"num_workers": 1, "pin_memory": True, "shuffle": False}
        train_kwargs.update(cuda_kwargs)
        test_kwargs.update(cuda_kwargs)

    transform = transforms.Compose(
        [transforms.ToTensor(), transforms.Normalize((0.1307,), (0.3081,))]
    )

    dataset1 = datasets.MNIST("../data", train=True, download=True, transform=transform)
    dataset2 = datasets.MNIST("../data", train=False, transform=transform)

    # NEW: Shard data
    sampler1 = DistributedSampler(
        dataset1, num_replicas=core_context.distributed.size, rank=core_context.distributed.rank
    )
    sampler2 = DistributedSampler(
        dataset2, num_replicas=core_context.distributed.size, rank=core_context.distributed.rank
    )
    train_loader = torch.utils.data.DataLoader(dataset1, sampler=sampler1, **train_kwargs)
    test_loader = torch.utils.data.DataLoader(dataset2, sampler=sampler2, **test_kwargs)

    hparams = info.trial.hparams

    # Wrap model with DDP. Aggregates gradients and synchronizes model training across slots
    model = Net(hparams).to(device)
    model = DDP(model, device_ids=[device], output_device=device)

    optimizer = optim.Adadelta(model.parameters(), lr=hparams["learning_rate"])
    scheduler = StepLR(optimizer, step_size=1, gamma=args.gamma)

    starting_epoch = 0
    epoch = starting_epoch
    last_checkpoint_batch = None

    for op in core_context.searcher.operations():

        while epoch < op.length:

            steps_completed = epoch * len(train_loader)
            train(args, model, device, train_loader, optimizer, core_context, epoch, op)
            test_loss = test(args, model, device, test_loader, core_context, steps_completed, op)

            scheduler.step()
            if args.save_model:

                checkpoint_metadata_dict = {
                    "steps_completed": steps_completed,
                }

            epoch += 1

            # Store checkpoints only on rank 0
            if core_context.distributed.rank == 0:
                with core_context.checkpoint.store_path(checkpoint_metadata_dict) as (
                    path,
                    storage_id,
                ):
                    torch.save(model.state_dict(), str(path) + ("/checkpoint.pt"))

            if core_context.preempt.should_preempt():
                return

        # Report completed only on rank 0
        if core_context.distributed.rank == 0:
            op.report_completed(test_loss)


if __name__ == "__main__":
    # NEW: Initialize process group using torch
    dist.init_process_group("nccl")

    # NEW: Initialize distributed context using from_torch_distributed
    # (obtains info such as rank, size, etc. from default torch environment variables)
    distributed = det.core.DistributedContext.from_torch_distributed()
    with det.core.init(distributed=distributed) as core_context:
        main(core_context)
