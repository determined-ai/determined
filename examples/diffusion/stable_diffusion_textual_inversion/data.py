import os
import PIL
from PIL import Image
from typing import Sequence

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
    """Dataset for textual inversion, pairing tokenized captions with images."""

    def __init__(
        self,
        train_img_dirs: Sequence[str],
        tokenizer: nn.Module,
        placeholder_tokens: Sequence[str],
        learnable_properties: Sequence[str],
        img_size: int = 512,
        interpolation: str = "bicubic",
        flip_p: float = 0.5,
        center_crop: bool = False,
    ):
        assert (
            len(train_img_dirs) == len(placeholder_tokens) == len(learnable_properties)
        ), "train_img_dirs, placeholder_tokens, and learnable_properties must have equal lens."

        assert (
            interpolation in INTERPOLATION_DICT
        ), f"interpolation must be in {list(INTERPOLATION_DICT.keys())}"
        self.interpolation = INTERPOLATION_DICT[interpolation]
        for prop in learnable_properties:
            assert (
                property in TEMPLATE_DICT
            ), f"learnable_properties must be one of {list(TEMPLATE_DICT.keys())}, not {prop}."

        self.train_img_dirs = train_img_dirs
        self.tokenizer = tokenizer
        self.learnable_properties = learnable_properties
        self.img_size = img_size
        self.placeholder_tokens = placeholder_tokens
        self.center_crop = center_crop
        self.flip_p = flip_p

        self.flip_transform = transforms.RandomHorizontalFlip(p=self.flip_p)

        # Create a dictionary of all images and their corresponding templates.
        self.img_dict = {}
        self.imgs = []
        self.records = 0
        for dir_path, learnable_property in zip(self.train_img_dirs, self.learnable_properties):
            imgs = []
            for file_path in os.listdir(dir_path):
                path = os.path.join(dir_path, file_path)
                try:
                    img = Image.open(path)
                    if not img.mode == "RGB":
                        img = img.convert("RGB")
                    imgs.append(img)
                except PIL.UnidentifiedImageError:
                    print(f"Image at {path} raised UnidentifiedImageError")
            template = TEMPLATE_DICT[learnable_property]
            self.img_dict["dir_path"] = {
                "template": template,
                "imgs": imgs,
            }
            self.records += len(imgs) * len(template)

    def __len__(self):
        return self.records

    def __getitem__(self, idx):
        # Generate a random caption drawn from the templates and include in the example.
        dir_idx, remainder = divmod(idx, self.train_img_dirs)
        dir_dict = self.img_dict[self.train_img_dirs[dir_idx]]
        template, imgs = dir_dict["template"], dir_dict["imgs"]
        placeholder_string = self.placeholder_tokens[dir_idx]

        template_idx, img_idx = divmod(remainder, len(template))
        text = template[template_idx].format(placeholder_string)
        img = imgs[img_idx]

        # Add text to example.
        example = {}
        example["input_ids"] = self.tokenizer(
            text,
            padding="max_length",
            truncation=True,
            max_length=self.tokenizer.model_max_length,
            return_tensors="pt",
        ).input_ids[0]

        # Add the corresponding normalized img tensor to the example.
        img_t = transforms.ToTensor()(img)
        if self.center_crop:
            crop_size = min(img_t.shape[-1], img_t.shape[-2])
            img_t = transforms.CenterCrop(crop_size)(img_t)
        img_t = transforms.Resize((self.img_size, self.img_size), interpolation=self.interpolation)(
            img_t
        )
        # Normalize the tensor to be in the range [-1, 1]
        img_t = (img_t - 0.5) * 2.0
        example["pixel_values"] = img_t

        return example
