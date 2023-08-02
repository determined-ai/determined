# This script reports training and testing losses as metrics
# to the Determined master via a Determined core.Context.
# This allows you to view metrics in the WebUI.

from __future__ import print_function

import argparse

import torch
import torch.nn as nn
import torch.nn.functional as F
import torch.optim as optim
from torch.optim.lr_scheduler import StepLR
from torchvision import datasets, transforms

# NEW: Import Determined.
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


# NEW: Modify function header to include core_context for metric
# reporting.
def train(args, model, device, train_loader, optimizer, epoch_idx, core_context):
    model.train()
    for batch_idx, (data, target) in enumerate(train_loader):
        data, target = data.to(device), target.to(device)
        optimizer.zero_grad()
        output = model(data)
        loss = F.nll_loss(output, target)
        loss.backward()
        optimizer.step()

        # NEW: Print training progress and loss at specified intervals
        # starting from the first batch.
        batches_completed = batch_idx + 1
        if batches_completed % args.log_interval == 0:
            print(
                "Train Epoch: {} [{}/{} ({:.0f}%)]\tLoss: {:.6f}".format(
                    epoch_idx,
                    batch_idx * len(data),
                    len(train_loader.dataset),
                    100.0 * batch_idx / len(train_loader),
                    loss.item(),
                )
            )
            # Docs snippet start: report training metrics
            # NEW: Report training metrics to Determined
            # master via core_context.
            # Index by (batch_idx + 1) * (epoch-1) * len(train_loader)
            # to continuously plot loss on one graph for consecutive
            # epochs.
            core_context.train.report_training_metrics(
                steps_completed=batches_completed + epoch_idx * len(train_loader),
                metrics={"train_loss": loss.item()},
            )
            # Docs snippet end: report training metrics
            if args.dry_run:
                break


# Docs snippet start: include args
# NEW: Modify function header to include args, epoch, test_loader,
# core_context for metric reporting and a steps_completed parameter to
# plot metrics.
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
    # Docs snippet end: include args
    # Docs snippet start: report validation metrics
    # NEW: Report validation metrics to Determined master
    # via core_context.
    core_context.train.report_validation_metrics(
        steps_completed=steps_completed,
        metrics={"test_loss": test_loss},
    )
    # Docs snippet end: report validation metrics


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

    ## Docs snippet start: log interval
    # NEW: Change log interval to 100 to reduce network overhead.
    parser.add_argument(
        "--log-interval",
        type=int,
        default=100,
        metavar="N",
        help="how many batches to wait before logging training status",
    )
    # Docs snippet end: log interval
    # Docs snippet start: remove save model
    # NEW: Remove save_model arg since this example only runs on
    # Determined and we do not need it for model checkpointing as
    # shown in later stages.

    args = parser.parse_args()
    use_cuda = not args.no_cuda and torch.cuda.is_available()
    use_mps = not args.no_mps and torch.backends.mps.is_available()

    torch.manual_seed(args.seed)
    # Docs snippet end: remove save model

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

    # NEW: change dataset1 to train_dataset and dataset_2 to test_dataset
    train_dataset = datasets.MNIST("../data", train=True, download=True, transform=transform)
    test_dataset = datasets.MNIST("../data", train=False, transform=transform)
    train_loader = torch.utils.data.DataLoader(train_dataset, **train_kwargs)
    test_loader = torch.utils.data.DataLoader(test_dataset, **test_kwargs)

    model = Net().to(device)
    optimizer = optim.Adadelta(model.parameters(), lr=args.lr)

    scheduler = StepLR(optimizer, step_size=1, gamma=args.gamma)
    for epoch_idx in range(0, args.epochs):
        # Docs snippet start: calculate steps completed
        # NEW: Calculate steps_completed for plotting test metrics.
        steps_completed = epoch_idx * len(train_loader)
        # Docs snippet end: calculate steps completed

        # Docs snippet start: pass core context
        # NEW: Pass core_context into train() and test().
        train(args, model, device, train_loader, optimizer, epoch_idx, core_context)

        # NEW: Pass args, test_loader, epoch, and steps_completed into
        # test().
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
        # Docs snippet end: pass core context

        # NEW: Remove model saving logic, checkpointing shown in next
        # stage.


# Docs snippet start: modify main loop core context
if __name__ == "__main__":
    # NEW: Establish new determined.core.Context and pass to main
    # function.
    with det.core.init() as core_context:
        main(core_context=core_context)
# Docs snippet end: modify main loop core content
