# The class MNISTDataModule is modified from the Pytorch Lightning example:
# https://colab.research.google.com/github/PytorchLightning/pytorch-lightning/
# blob/master/notebooks/01-mnist-hello-world.ipynb#scrollTo=4DNItffri95Q
#
# Copyright The PyTorch Lightning team.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
import os
import urllib.parse
import requests
import logging
import shutil
from torchvision import transforms
from torch.utils.data import DataLoader, random_split
import pytorch_lightning as pl
from typing import Optional
from torchvision.datasets import MNIST

class MNISTDataModule(pl.LightningDataModule):
    def __init__(self, data_url: str, data_dir: str = '/tmp/det', batch_size=64):
        super().__init__()
        self.data_dir = data_dir
        self.batch_size = batch_size
        self.data_url = data_url
        # self.dims is returned when you call dm.size()
        # Setting default dims here because we know them.
        # Could optionally be assigned dynamically in dm.setup()
        self.dims = (1, 28, 28)


    def prepare_data(self):
        # prepare transforms standard to MNIST
        url_path = urllib.parse.urlparse(self.data_url).path
        basename = url_path.rsplit("/", 1)[1]

        download_directory = os.path.join(self.data_dir, "MNIST")
        os.makedirs(download_directory, exist_ok=True)
        filepath = os.path.join(download_directory, basename)
        if not os.path.exists(filepath):
            logging.info("Downloading {} to {}".format(self.data_url, filepath))

            r = requests.get(self.data_url, stream=True)
            with open(filepath, "wb") as f:
                for chunk in r.iter_content(chunk_size=8192):
                    if chunk:
                        f.write(chunk)

        shutil.unpack_archive(filepath, download_directory)

        self.data_dir = os.path.dirname(download_directory)

    def setup(self, stage: Optional[str] = None):
        transform = transforms.Compose([transforms.ToTensor(),
                                        transforms.Normalize((0.1307,), (0.3081,))])

        # Assign train/val datasets for use in dataloaders
        if stage == 'fit' or stage is None:
            mnist_full = MNIST(self.data_dir, train=True, transform=transform)
            self.mnist_train, self.mnist_val = random_split(mnist_full, [55000, 5000])

        # Assign test dataset for use in dataloader(s)
        if stage == 'test' or stage is None:
            self.mnist_test = MNIST(self.data_dir, train=False, transform=transform)


    def train_dataloader(self):
        return DataLoader(self.mnist_train, batch_size=self.batch_size)
    def val_dataloader(self):
        return DataLoader(self.mnist_val, batch_size=self.batch_size)
