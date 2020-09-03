import numpy as np
from torch.utils.data import Dataset


class BilevelDataset(Dataset):
    def __init__(
        self,
        dataset,
    ):
        """
        We will split the data into a train split and a validation split
        and return one image from each split as a single observation.

        Args:
            dataset: PyTorch Dataset object
        """
        inds = np.arange(len(dataset))
        self.dataset = dataset
        # Make sure train and val splits are of equal size.
        # This is so we make sure to loop images in both train
        # and val splits exactly once in an epoch.
        n_train = int(0.5 * len(inds))
        self.train_inds = inds[0:n_train]
        self.val_inds = inds[n_train : 2 * n_train]
        assert len(self.train_inds) == len(self.val_inds)

    def shuffle_val_inds(self):
        # This is so we will see different pairs of images
        # from train and val splits.  Will need to call this
        # manually at epoch end.
        np.random.shuffle(self.val_inds)

    def __len__(self):
        return len(self.train_inds)

    def __getitem__(self, idx):
        train_ind = self.train_inds[idx]
        val_ind = self.val_inds[idx]
        x_train, y_train = self.dataset[train_ind]
        x_val, y_val = self.dataset[val_ind]
        return x_train, y_train, x_val, y_val
