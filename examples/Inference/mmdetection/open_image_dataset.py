import os

from torch.utils.data import Dataset


class OpenImageDataset(Dataset):
    def __init__(self, img_dir):
        self.data = []
        # Iterate directory
        for root, dirs, files in os.walk(img_dir, topdown=False):
            print(root)
            for name in files:
                if name.endswith(".jpg"):
                    self.data.append(os.path.join(root, name))

    def __len__(self):
        return len(self.data)

    def __getitem__(self, idx):
        return self.data[idx]
