import logging
import os
import PIL
from pathlib import Path
from PIL import Image
from typing import List, Tuple, Sequence

import torch
from torch.utils.data import Dataset
from torchvision import transforms

from detsd import defaults


class TextualInversionDataset(Dataset):
    """Dataset for textual inversion, pairing prompts with normalized image tensors."""

    def __init__(
        self,
        img_dirs: Sequence[str],
        concept_strs: Sequence[str],
        learnable_properties: Sequence[str],
        img_size: int = 512,
        interpolation: str = "bicubic",
        flip_p: float = 0.0,
        center_crop: bool = False,
        append_file_name_to_text: bool = False,
        file_name_split_char: str = "_",
        num_blank_prompts: int = 100,
        num_a_prompts: int = 100,
    ):
        assert (
            len(img_dirs) == len(concept_strs) == len(learnable_properties)
        ), "img_dirs, concept_strs, and learnable_properties must have equal lens."

        assert (
            interpolation in defaults.INTERPOLATION_DICT
        ), f"interpolation must be in {list(defaults.INTERPOLATION_DICT.keys())}"
        self.interpolation = defaults.INTERPOLATION_DICT[interpolation]
        for prop in learnable_properties:
            assert (
                prop in defaults.TEMPLATE_DICT
            ), f"learnable_properties must be one of {list(defaults.TEMPLATE_DICT.keys())}, not {prop}."

        self.img_dirs = img_dirs
        self.learnable_properties = learnable_properties
        self.img_size = img_size
        self.concept_strs = concept_strs
        self.center_crop = center_crop
        self.flip_p = flip_p
        self.append_file_name_to_text = append_file_name_to_text
        self.file_name_split_char = file_name_split_char

        self._base_img_trans = transforms.Compose(
            [
                transforms.Resize(size=self.img_size, interpolation=self.interpolation),
                transforms.ToTensor(),
                transforms.RandomHorizontalFlip(p=self.flip_p),
            ]
        )

        self.logger = logging.getLogger(__name__)

        self.records = []
        for dir_path, concept_str, prop in zip(
            self.img_dirs, concept_strs, self.learnable_properties
        ):
            templates = defaults.TEMPLATE_DICT[prop]
            templates.extend(["{}"] * num_blank_prompts)
            templates.extend(["a {}"] * num_a_prompts)
            imgs_and_paths = self._get_imgs_and_paths_from_dir_path(dir_path)
            for img, path in imgs_and_paths:
                img_tensor = self._convert_img_to_tensor(img)
                for text in templates:
                    prompt = text.format(concept_str)
                    if append_file_name_to_text:
                        file_name_without_extension = path.stem
                        split_file_name = file_name_without_extension.split(
                            self.file_name_split_char
                        )
                        joined_file_name = " ".join(split_file_name)
                        prompt = f"{prompt} {joined_file_name}"
                    self.records.append((prompt, img_tensor))

    def _get_imgs_and_paths_from_dir_path(self, dir_path: str) -> List[Tuple[Image.Image, Path]]:
        """Returns a list of PIL Images loaded from all valid files contained in dir_path."""
        imgs_and_paths = []
        for file_or_dir in os.listdir(dir_path):
            path = Path(dir_path).joinpath(file_or_dir)
            if path.is_file():
                try:
                    img = Image.open(path)
                    if not img.mode == "RGB":
                        img = img.convert("RGB")
                    imgs_and_paths.append((img, path))
                except PIL.UnidentifiedImageError:
                    self.logger.warning(
                        f"File at {path} raised UnidentifiedImageError and will be skipped."
                    )
        return imgs_and_paths

    def _convert_img_to_tensor(self, img: Image.Image) -> torch.Tensor:
        """Converts a PIL image into an appropriately transformed tensor."""
        if self.center_crop:
            img = transforms.CenterCrop(size=min(img.size))(img)
        img_tensor = self._base_img_trans(img)
        # Normalize the tensor to be in the range [-1, 1]
        img_tensor = (img_tensor - 0.5) * 2.0
        return img_tensor

    def __len__(self):
        return len(self.records)

    def __getitem__(self, idx) -> Tuple[str, torch.Tensor]:
        return self.records[idx]
