import logging
import os
import shutil
import urllib.parse

import requests

import pytorch_lightning as pl
from torchvision import transforms
from torch.utils.data import DataLoader, random_split
from torchvision.datasets import MNIST


class MNISTDataModule(pl.LightningDataModule):

    def __init__(self, data_url: str = '', data_dir: str = './', batch_size: int = 64, num_workers: int = 8):
        super().__init__()
        self.data_url = data_url
        self.data_dir = data_dir
        self.batch_size = batch_size
        self.num_workers = num_workers

        self.transform = transforms.Compose([
            transforms.ToTensor(),
            transforms.Normalize((0.1307,), (0.3081,))
        ])

        # self.dims is returned when you call dm.size()
        # Setting default dims here because we know them.
        # Could optionally be assigned dynamically in dm.setup()
        self.dims = (1, 28, 28)
        self.num_classes = 10

    def prepare_data(self):
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

    def setup(self, stage=None):

        # Assign train/val datasets for use in dataloaders
        if stage == 'fit' or stage is None:
            mnist_full = MNIST(self.data_dir, train=True, transform=self.transform)
            self.mnist_train, self.mnist_val = random_split(mnist_full, [55000, 5000])

        # Assign test dataset for use in dataloader(s)
        if stage == 'test' or stage is None:
            self.mnist_test = MNIST(self.data_dir, train=False, transform=self.transform)

    def train_dataloader(self):
        return DataLoader(self.mnist_train, batch_size=self.batch_size, num_workers=self.num_workers)

    def val_dataloader(self):
        return DataLoader(self.mnist_val, batch_size=self.batch_size, num_workers=self.num_workers)

    def test_dataloader(self):
        return DataLoader(self.mnist_test, batch_size=self.batch_size, num_workers=self.num_workers)

