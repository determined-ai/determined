import pytorch_lightning as ptl
from determined.pytorch import DataLoader as DetDataLoader
from torch.utils.data import DataLoader

class DETLightningDataModule(ptl.LightningDataModule):
    """
    ## user defines these as usual
    def prepare_data
        # download, split, etc...
        # only called on 1 GPU/TPU in distributed

    def setup
        # in memory assignments. called on every process in DDP

    ## user defines these for DET usage. similar to normal ptl datamodule
    def train_det_dataloader: DetDataloader
    def val_det_dataloader: DetDataloader
    def test_det_dataloader: DetDataloader

    ## user gets these for free
    def train_dataloader
    def val_dataloader
    def test_dataloader
    """
    def __init__(self, *args, **kwargs):
        return super().__init__(*args, **kwargs)


    def train_det_dataloader(self) -> DetDataLoader:
        raise NotImplementedError

    def val_det_dataloader(self) -> DetDataLoader:
        raise NotImplementedError

    def train_dataloader(self) -> DataLoader:
        return self.train_det_dataloader().get_data_loader()

    def val_dataloader(self) -> DataLoader:
        return self.train_det_dataloader().get_data_loader()

    # def test_dataloader(self) -> DataLoader:
    #     raise TypeError('not supported')
