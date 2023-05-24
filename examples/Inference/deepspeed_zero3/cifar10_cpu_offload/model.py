import torch.nn.functional as F
from torch.nn import Conv2d, CrossEntropyLoss, Linear, MaxPool2d, Module


class Net(Module):
    def __init__(self):
        super(Net, self).__init__()

        self.conv1_1 = Conv2d(3, 1024, kernel_size=3, padding=1)
        self.conv1_2 = Conv2d(1024, 2048, kernel_size=3, stride=1, padding=1)
        self.pool = MaxPool2d(2, 2)
        self.conv2_1 = Conv2d(2048, 3072, kernel_size=3, stride=1, padding=1)
        self.conv2_2 = Conv2d(3072, 4096, kernel_size=3, stride=1, padding=1)
        self.fc1 = Linear(4096 * 8 * 8, 1000)
        self.fc2 = Linear(1000, 10000)
        self.fc3 = Linear(10000, 10000)
        self.fc4 = Linear(10000, 10)

    def forward(self, x):
        x = self.pool(F.relu(self.conv1_2(F.relu(self.conv1_1(x)))))
        x = self.pool(F.relu(self.conv2_2(F.relu(self.conv2_1(x)))))
        x = x.view(-1, 4096 * 8 * 8)
        x = F.relu(self.fc1(x))
        x = F.relu(self.fc2(x))
        x = F.relu(self.fc3(x))
        x = self.fc4(x)
        return x


def get_model():
    return Net()
