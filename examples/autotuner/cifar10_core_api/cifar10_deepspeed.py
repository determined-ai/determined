import uuid

import numpy as np
import torch
import torch.nn as nn
import torch.nn.functional as F
import torch.optim as optim
import torchvision
import torchvision.transforms as transforms
from attrdict import AttrDict
from deepspeed.moe.utils import split_params_into_different_moe_groups_for_optimizer

import deepspeed
import determined as det
from determined.pytorch.deepspeed import overwrite_deepspeed_config


def main(args, info, context):
    deepspeed.init_distributed()

    ########################################################################
    # The output of torchvision datasets are PILImage images of range [0, 1].
    # We transform them to Tensors of normalized range [-1, 1].
    # .. note::
    #     If running on Windows and you get a BrokenPipeError, try setting
    #     the num_worker of torch.utils.data.DataLoader() to 0.

    transform = transforms.Compose(
        [transforms.ToTensor(), transforms.Normalize((0.5, 0.5, 0.5), (0.5, 0.5, 0.5))]
    )

    if torch.distributed.get_rank() != 0:
        # might be downloading cifar data, let rank 0 download first
        torch.distributed.barrier()

    trainset = torchvision.datasets.CIFAR10(
        root="./data", train=True, download=True, transform=transform
    )

    if torch.distributed.get_rank() == 0:
        # cifar data is downloaded, indicate other ranks can proceed
        torch.distributed.barrier()

    trainloader = torch.utils.data.DataLoader(trainset, batch_size=16, shuffle=True, num_workers=2)

    testset = torchvision.datasets.CIFAR10(
        root="./data", train=False, download=True, transform=transform
    )
    testloader = torch.utils.data.DataLoader(testset, batch_size=4, shuffle=False, num_workers=2)

    classes = ("plane", "car", "bird", "cat", "deer", "dog", "frog", "horse", "ship", "truck")

    # get some random training images
    dataiter = iter(trainloader)
    images, labels = dataiter.next()

    # print labels
    print(" ".join("%5s" % classes[labels[j]] for j in range(4)))

    ########################################################################
    # 2. Define a Convolutional Neural Network
    # ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
    # Copy the neural network from the Neural Networks section before and modify it to
    # take 3-channel images (instead of 1-channel images as it was defined).

    class Net(nn.Module):
        def __init__(self):
            super(Net, self).__init__()
            self.conv1 = nn.Conv2d(3, 6, 5)
            self.pool = nn.MaxPool2d(2, 2)
            self.conv2 = nn.Conv2d(6, 16, 5)
            self.fc1 = nn.Linear(16 * 5 * 5, 120)
            self.fc2 = nn.Linear(120, 84)
            if args.moe:
                fc3 = nn.Linear(84, 84)
                self.moe_layer_list = []
                for n_e in args.num_experts:
                    # create moe layers based on the number of experts
                    self.moe_layer_list.append(
                        deepspeed.moe.layer.MoE(
                            hidden_size=84,
                            expert=fc3,
                            num_experts=n_e,
                            ep_size=args.ep_world_size,
                            use_residual=args.mlp_type == "residual",
                            k=args.top_k,
                            min_capacity=args.min_capacity,
                            noisy_gate_policy=args.noisy_gate_policy,
                        )
                    )
                self.moe_layer_list = nn.ModuleList(self.moe_layer_list)
                self.fc4 = nn.Linear(84, 10)
            else:
                self.fc3 = nn.Linear(84, 10)

        def forward(self, x):
            x = self.pool(F.relu(self.conv1(x)))
            x = self.pool(F.relu(self.conv2(x)))
            x = x.view(-1, 16 * 5 * 5)
            x = F.relu(self.fc1(x))
            x = F.relu(self.fc2(x))
            if args.moe:
                for layer in self.moe_layer_list:
                    x, _, _ = layer(x)
                x = self.fc4(x)
            else:
                x = self.fc3(x)
            return x

    net = Net()

    def create_moe_param_groups(model):
        parameters = {"params": [p for p in model.parameters()], "name": "parameters"}

        return split_params_into_different_moe_groups_for_optimizer(parameters)

    parameters = filter(lambda p: p.requires_grad, net.parameters())
    if args.moe_param_group:
        parameters = create_moe_param_groups(net)

    # Initialize DeepSpeed to use the following features
    # 1) Distributed model
    # 2) Distributed data loader
    # 3) DeepSpeed optimizer
    ds_config_file = args.pop("deepspeed_config")
    ds_config = det.pytorch.deepspeed.overwrite_deepspeed_config(
        ds_config_file, {"optimizer": {"params": {"lr": args.learning_rate}}}
    )

    model_engine, optimizer, trainloader, __ = deepspeed.initialize(
        model=net, model_parameters=parameters, training_data=trainset, config=ds_config
    )

    fp16 = model_engine.fp16_enabled()
    print(f"fp16={fp16}")

    # device = torch.device("cuda:0" if torch.cuda.is_available() else "cpu")
    # net.to(device)
    ########################################################################
    # 3. Define a Loss function and optimizer
    # ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
    # Let's use a Classification Cross-Entropy loss and SGD with momentum.

    criterion = nn.CrossEntropyLoss()
    # optimizer = optim.SGD(net.parameters(), lr=0.001, momentum=0.9)

    ########################################################################
    # 4. Train the network
    # ^^^^^^^^^^^^^^^^^^^^
    #
    # This is when things start to get interesting.
    # We simply have to loop over our data iterator, and feed the inputs to the
    # network and optimize.
    step = 0
    epochs_trained = 0
    if info.latest_checkpoint is not None:
        with context.checkpoint.restore_path(info.latest_checkpoint) as path:
            _, client_state = model_engine.load_checkpoint(path, tag="cifar10")
            step = client_state["latests_batch"]
            epochs_trained = client_state["epochs_trained"]
    for op in context.searcher.operations():
        while epochs_trained < op.length:
            running_loss = 0.0
            for i, data in enumerate(trainloader):
                # get the inputs; data is a list of [inputs, labels]
                inputs, labels = data[0].to(model_engine.local_rank), data[1].to(
                    model_engine.local_rank
                )
                if fp16:
                    inputs = inputs.half()
                outputs = model_engine(inputs)
                loss = criterion(outputs, labels)

                model_engine.backward(loss)
                model_engine.step()

                # print statistics
                running_loss += loss.item()
                step += 1
                if i % args.log_interval == (
                    args.log_interval - 1
                ):  # print every log_interval mini-batches
                    print(
                        "[%d, %5d] loss: %.3f"
                        % (epochs_trained + 1, i + 1, running_loss / args.log_interval)
                    )
                    running_loss = 0.0
                    if distributed.rank == 0:
                        context.train.report_training_metrics(
                            steps_completed=step, metrics={"loss": loss.item()}
                        )
            epochs_trained += 1
            if context.distributed.rank == 0:
                op.report_progress(epochs_trained)

        metadata = {"steps_completed": step}
        storage_manager = context.checkpoint._storage_manager
        storage_id = str(uuid.uuid4())
        if context.distributed.rank == 0:
            with storage_manager.store_path(storage_id) as path:
                # Broadcast checkpoint path to all ranks.
                context.distributed.broadcast((storage_id, path))
                model_engine.save_checkpoint(
                    path,
                    tag="cifar10",
                    client_state={"steps_completed": step, "epochs_trained": epochs_trained},
                )
                # Gather resources across nodes.
                all_resources = context.distributed.gather(storage_manager._list_directory(path))
            resources = {k: v for d in all_resources for k, v in d.items()}

            context.checkpoint._report_checkpoint(storage_id, resources, metadata)
        else:
            storage_id, path = context.distributed.broadcast(None)
            model_engine.save_checkpoint(path, tag="cifar10", client_state=metadata)
            # Gather resources across nodes.
            _ = context.distributed.gather(storage_manager._list_directory(path))
            if context.distributed.local_rank == 0:
                storage_manager.post_store_path(storage_id, path)
        correct = 0
        total = 0
        with torch.no_grad():
            for data in testloader:
                images, labels = data
                if fp16:
                    images = images.half()
                outputs = net(images.to(model_engine.local_rank))
                _, predicted = torch.max(outputs.data, 1)
                total += labels.size(0)
                correct += (predicted == labels.to(model_engine.local_rank)).sum().item()
        accuracy = 100 * correct / total

        print("Accuracy of the network on the 10000 test images: %d %%" % accuracy)

        metrics = {"accuracy": accuracy}
        if distributed.rank == 0:
            context.train.report_validation_metrics(steps_completed=step, metrics=metrics)
            op.report_completed(accuracy)


if __name__ == "__main__":
    distributed = det.core.DistributedContext.from_deepspeed()
    info = det.get_cluster_info()
    args = AttrDict(info.trial.hparams)

    with det.core.init(distributed=distributed) as context:
        main(args, info, context)
