import os
import urllib.parse
import requests
import logging
import shutil
from torchvision import transforms
from torch.utils.data import DataLoader, random_split
import pytorch_lightning as pl
from typing import Optional

# TorchGeo specific
from torchgeo.datasets import NAIP, ChesapeakeDE
from torchgeo.datasets.utils import download_url

class NAIPDataModule(pl.LightningDataModule):
    def __init__(self, data_dir: str = '/tmp/det', batch_size=64):
        super().__init__()
        self.data_dir = data_dir
        self.batch_size = batch_size
        # self.dims is returned when you call dm.size()
        # Setting default dims here because we know them.
        # Could optionally be assigned dynamically in dm.setup()
        self.dims = (4, 1667, 1667)

    def prepare_data(self):
        # prepare transforms
        download_directory = os.path.join(self.data_dir, "NAIP")
        os.makedirs(download_directory, exist_ok=True)
        self.naip_root = download_directory

        naip_url = "https://naipblobs.blob.core.windows.net/naip/v002/de/2018/de_060cm_2018/38075/"
        tiles = [
            "m_3807511_ne_18_060_20181104.tif",
            "m_3807511_se_18_060_20181104.tif",
            "m_3807512_nw_18_060_20180815.tif",
            "m_3807512_sw_18_060_20180815.tif",
        ]
        for tile in tiles:
            download_url(naip_url + tile, download_directory)

        self.data_dir = os.path.dirname(download_directory)

    def setup(self, stage: Optional[str] = None):
        transform = transforms.Compose([transforms.ToTensor(),
                                        transforms.Normalize((0.1307,), (0.3081,))])

        naip = NAIP(self.naip_root)
        chesapeake_root = os.path.join(self.data_dir, "chesapeake")
        chesapeake = ChesapeakeDE(chesapeake_root, crs=naip.crs, res=naip.res, download=True)
        self.dataset = naip + chesapeake

    def train_dataloader(self):
        return DataLoader(self.dataset, batch_size=self.batch_size)
    def val_dataloader(self):
        return DataLoader(self.dataset, batch_size=self.batch_size)
