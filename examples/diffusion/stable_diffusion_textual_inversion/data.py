import os
import PIL
import random
from PIL import Image

import numpy as np
import torch.nn as nn
from torch.utils.data import Dataset
from torchvision import transforms

from training_templates import TEMPLATE_DICT

INTERPOLATION_DICT = {
    "nearest": transforms.InterpolationMode.NEAREST,
    "bilinear": transforms.InterpolationMode.BILINEAR,
    "bicubic": transforms.InterpolationMode.BICUBIC,
}

MAX_INT = 2 ** 32 - 1


class TextualInversionDataset(Dataset):
    """Create an effectively infinite dataset. The Dataset's __getitem__ method returns a dictionary with
    input_ids and pixel_values keys, where input_ids come from applying the tokenizer to a caption
    describing the img (randomly drawn from fixed templates) and pixel_values are the normalized
    tensor values of the img."""

    def __init__(
        self,
        train_img_dir: str,
        tokenizer: nn.Module,
        placeholder_token: str,
        learnable_property: str = "object",
        size: int = 512,
        interpolation: str = "bicubic",
        flip_p: float = 0.5,
        center_crop: bool = False,
    ):

        self.train_img_dir = train_img_dir
        self.tokenizer = tokenizer
        self.learnable_property = learnable_property
        self.size = size
        self.placeholder_token = placeholder_token
        self.center_crop = center_crop
        self.flip_p = flip_p

        self.img_paths = [
            os.path.join(self.train_img_dir, file_path)
            for file_path in os.listdir(self.train_img_dir)
        ]
        self.num_imgs = len(self.img_paths)

        assert (
            interpolation in INTERPOLATION_DICT
        ), f"interpolation must be in {list(INTERPOLATION_DICT.keys())}"
        self.interpolation = INTERPOLATION_DICT[interpolation]

        assert (
            learnable_property in TEMPLATE_DICT
        ), f"learnable_property must be one of {list(TEMPLATE_DICT.keys())}"

        self.templates = TEMPLATE_DICT[learnable_property]
        self.num_templates = len(self.templates)
        self.flip_transform = transforms.RandomHorizontalFlip(p=self.flip_p)

    def __len__(self):
        return self.num_imgs * self.num_templates

    def __getitem__(self, idx):
        template_idx, img_idx = divmod(idx, self.num_imgs)
        example = {}

        # Generate a random caption drawn from the templates and include in the example.
        placeholder_string = self.placeholder_token
        text = self.templates[template_idx].format(placeholder_string)
        example["input_ids"] = self.tokenizer(
            text,
            padding="max_length",
            truncation=True,
            max_length=self.tokenizer.model_max_length,
            return_tensors="pt",
        ).input_ids[0]

        # Add the corresponding normalized img tensor to the example.
        img = Image.open(self.img_paths[img_idx])
        if not img.mode == "RGB":
            img = img.convert("RGB")
        img_t = transforms.ToTensor()(img)
        if self.center_crop:
            crop_size = min(img_t.shape[-1], img_t.shape[-2])
            img_t = transforms.CenterCrop(crop_size)(img_t)
        img_t = transforms.Resize((self.size, self.size), interpolation=self.interpolation)(img_t)
        # Normalize the tensor to be in the range [-1, 1]
        img_t = (img_t - 0.5) * 2.0
        example["pixel_values"] = img_t

        return example
