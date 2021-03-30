# Each training example will be a classification task with n classes.

import os
import numpy as np

import torch
from torch.utils.data import Dataset

from PIL import Image


class OmniglotTasks(Dataset):
    def __init__(
        self,
        data_path,
        tasks_per_epoch,
        class_idxs,
        img_resize_dim,
        num_classes,
        num_support,
        num_query=None,
        rotations=(0, 90, 180, 270),
    ):
        """
        Each subfolder of the parent data directory is a separate
        alphabet with multiple characters.  Each class is a certain
        character within an alphabet.  Tasks will correspond to
        n-way classification problems where the learner will
        have to predict which of n classes a test image belongs to.

        Args:
            # NOTE: You should partition all indices to a train set and a validation
            # set of indices and use those as args here.
            class_idxs: character indices to sample from to generate tasks.
            num_support: how many image per class to use to construct class prototypes
            num_query: how many images per class to use to update embedding

        """
        self.class_idxs = class_idxs
        self.rotations = rotations
        self.tasks_per_epoch = tasks_per_epoch
        self.img_resize_dim = img_resize_dim

        self.class_paths = {}
        class_idx = 0
        min_class_examples = float("inf")
        for root, dirs, files in os.walk(data_path):
            if len(dirs) == 0:
                min_class_examples = min(min_class_examples, len(files))
                self.class_paths[class_idx] = [os.path.join(root, f) for f in files]
                class_idx += 1

        self.num_classes = num_classes
        self.num_support = num_support
        self.num_query = (
            min_class_examples - num_support if (num_query is None) else num_query
        )

    def get_collate_fn(self):
        """
        This collate function returns a list of dictionaries in a batch.

        Whereas by default, the collate function zips the dictionary field
        values into a list and returns a single dictionary.
        """

        def collate(examples):
            return examples

        return collate

    def __len__(self):
        return self.tasks_per_epoch

    def __getitem__(self, idx):
        task_classes = np.random.choice(
            self.class_idxs, size=self.num_classes, replace=False
        )
        rotations = np.random.choice(
            self.rotations, size=self.num_classes, replace=True
        )

        imgs = []
        labels = []
        for i, cls in enumerate(task_classes):
            imgs_paths = np.random.choice(
                self.class_paths[cls],
                size=(self.num_support + self.num_query),
                replace=False,
            )
            for pth in imgs_paths:
                with open(pth, "rb") as f:
                    img = Image.open(f).resize(
                        (self.img_resize_dim, self.img_resize_dim)
                    )
                    if len(self.rotations):
                        rot = rotations[i]
                        img = img.rotate(rot)
                    img = np.array(img).astype(np.float32)
                    imgs.append(img[np.newaxis, :])
                    labels.append(i)
        support_idxs = [
            cls + i
            for i in range(self.num_support)
            for cls in range(0, len(labels), self.num_support + self.num_query)
        ]
        query_idxs = [
            cls + self.num_support + i
            for i in range(self.num_query)
            for cls in range(0, len(labels), self.num_support + self.num_query)
        ]

        x_support = np.stack([imgs[i] for i in support_idxs])
        y_support = np.array([labels[i] for i in support_idxs])
        x_query = np.stack([imgs[i] for i in query_idxs])
        y_query = np.array([labels[i] for i in query_idxs])
        return {
            "support": (torch.from_numpy(x_support), torch.from_numpy(y_support)),
            "query": (torch.from_numpy(x_query), torch.from_numpy(y_query)),
        }
