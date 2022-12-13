from typing import Dict

import numpy as np
from PIL import Image
from torch.utils.data import Dataset


class FakeParser:
    def __init__(self):
        self.img_ids = []

    def create_fake_img_ids(self, num_indices):
        self.img_ids = [np.random.randint(1, 90) for i in range(num_indices)]


class FakeBackend(Dataset):
    def __init__(self, transform=None):
        self.transform = transform

    def __len__(self):
        return 1000

    def __getitem__(self, i):
        target = dict(img_idx=i, img_size=(512, 512))

        img = Image.open("loss_by_gpus.png").convert("RGB")
        img = img.resize((512, 512))

        if self.transform is not None:
            img, target = self.transform(img, target)

        target["bbox"] = np.random.rand(2, 4)
        target["cls"] = np.array([np.random.randint(90), np.random.randint(90)])

        return img, target


class DotDict(dict):
    __setattr__ = dict.__setitem__
    __delattr__ = dict.__delitem__

    def __init__(self, dct):
        for key, value in dct.items():
            if value == "None":
                value = None
            self[key] = value

    def __getattr__(self, name):
        try:
            return self[name]
        except:
            return None
