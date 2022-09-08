import os
import PIL
import random
from PIL import Image

import numpy as np
import torch
from torch.utils.data import Dataset
from torchvision import transforms

from training_templates import IMAGEN_OBJECT_TEMPLATES_SMALL, IMAGEN_STYLE_TEMPLATES_SMALL

INTERPOLATION_DICT = {
    "nearest": transforms.InterpolationMode.NEAREST,
    "bilinear": transforms.InterpolationMode.BILINEAR,
    "bicubic": transforms.InterpolationMode.BICUBIC,
}


class TextualInversionDataset(Dataset):
    def __init__(
        self,
        data_root,
        tokenizer,
        learnable_property="object",  # [object, style]
        size=512,
        repeats=100,
        interpolation="bicubic",
        flip_p=0.5,
        split="train",
        placeholder_token="*",
        center_crop=False,
    ):

        self.data_root = data_root
        self.tokenizer = tokenizer
        self.learnable_property = learnable_property
        self.size = size
        self.placeholder_token = placeholder_token
        self.center_crop = center_crop
        self.flip_p = flip_p

        self.image_paths = [
            os.path.join(self.data_root, file_path) for file_path in os.listdir(self.data_root)
        ]

        self.num_images = len(self.image_paths)
        self._length = self.num_images

        if split == "train":
            self._length = self.num_images * repeats
        assert (
            interpolation in INTERPOLATION_DICT
        ), f"interpolation must be in {list(INTERPOLATION_DICT.keys())}"
        self.interpolation = INTERPOLATION_DICT[interpolation]

        assert learnable_property in (
            "object",
            "style",
        ), f'learnable_property must be "object" or "style", not {learnable_property}'

        self.templates = (
            IMAGEN_STYLE_TEMPLATES_SMALL
            if learnable_property == "style"
            else IMAGEN_OBJECT_TEMPLATES_SMALL
        )
        self.flip_transform = transforms.RandomHorizontalFlip(p=self.flip_p)

    def __len__(self):
        return self._length

    def __getitem__(self, i):
        example = {}

        # Add the tokenized input text to the example
        placeholder_string = self.placeholder_token
        text = random.choice(self.templates).format(placeholder_string)

        example["input_ids"] = self.tokenizer(
            text,
            padding="max_length",
            truncation=True,
            max_length=self.tokenizer.model_max_length,
            return_tensors="pt",
        ).input_ids[0]

        # Add the normalized image tensor to the example
        image = Image.open(self.image_paths[i % self.num_images])
        if not image.mode == "RGB":
            image = image.convert("RGB")
        image_t = transforms.ToTensor()(image)
        if self.center_crop:
            crop_size = min(image_t.shape[-1], image_t.shape[-2])
            image_t = transforms.CenterCrop(crop_size)(image_t)
        image_t = transforms.Resize(self.size, interpolation=self.interpolation)(image_t)
        # Normalize the tensor to be in the range [-1, 1]
        image_t = (image_t - 0.5) * 2.0
        example["pixel_values"] = image_t

        return example
