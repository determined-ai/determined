import os
import PIL
import torch
from PIL import Image
from typing import List, Tuple, Sequence

from torch.utils.data import Dataset
from torchvision import transforms

from training_templates import TEMPLATE_DICT

INTERPOLATION_DICT = {
    "nearest": transforms.InterpolationMode.NEAREST,
    "bilinear": transforms.InterpolationMode.BILINEAR,
    "bicubic": transforms.InterpolationMode.BICUBIC,
}


class TextualInversionDataset(Dataset):
    """Dataset for textual inversion, pairing captions with normalized image tensors.  The
    'input_text' and 'pixel_values' keys of each record correspond to the text and image tensor,
    respectively, with the latter normalized to lie in the range [-1, 1].
    """

    def __init__(
        self,
        train_img_dirs: Sequence[str],
        concept_tokens: Sequence[str],
        learnable_properties: Sequence[str],
        img_size: int = 512,
        interpolation: str = "bicubic",
        flip_p: float = 0.0,
        center_crop: bool = False,
        append_file_name_to_text: bool = False,
        file_name_split_char: str = "_",
    ):
        assert (
            len(train_img_dirs) == len(concept_tokens) == len(learnable_properties)
        ), "train_img_dirs, concept_tokens, and learnable_properties must have equal lens."

        assert (
            interpolation in INTERPOLATION_DICT
        ), f"interpolation must be in {list(INTERPOLATION_DICT.keys())}"
        self.interpolation = INTERPOLATION_DICT[interpolation]
        for prop in learnable_properties:
            assert (
                prop in TEMPLATE_DICT
            ), f"learnable_properties must be one of {list(TEMPLATE_DICT.keys())}, not {prop}."

        self.train_img_dirs = train_img_dirs
        self.learnable_properties = learnable_properties
        self.img_size = img_size
        self.concept_tokens = concept_tokens
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

        self.records = []
        for dir_path, concept_token, prop in zip(
            self.train_img_dirs, concept_tokens, self.learnable_properties
        ):
            templates = TEMPLATE_DICT[prop]
            imgs_and_file_paths = self._get_imgs_and_file_paths_from_dir_path(dir_path)
            for img, file_path in imgs_and_file_paths:
                img_t = self._convert_img_to_tensor(img)
                for text in templates:
                    text_with_token = text.format(concept_token)
                    if append_file_name_to_text:
                        file_path_without_extension = ".".join(file_path.split(".")[:-1])
                        split_file_name = file_path_without_extension.split(
                            self.file_name_split_char
                        )
                        joined_file_name = " ".join(split_file_name)
                        text_with_token = f"{text_with_token} {joined_file_name}"
                    self.records.append({"input_text": text_with_token, "pixel_values": img_t})

    def _get_imgs_and_file_paths_from_dir_path(
        self, dir_path: str
    ) -> List[Tuple[Image.Image, str]]:
        """Returns a list of PIL Images loaded from all valid files contained in dir_path."""
        imgs_and_file_paths = []
        for file_path in os.listdir(dir_path):
            path = os.path.join(dir_path, file_path)
            try:
                img = Image.open(path)
                if not img.mode == "RGB":
                    img = img.convert("RGB")
                imgs_and_file_paths.append((img, file_path))
            except PIL.UnidentifiedImageError:
                print(f"File at {path} raised UnidentifiedImageError and will be skipped.")
        return imgs_and_file_paths

    def _convert_img_to_tensor(self, img: Image.Image) -> torch.Tensor:
        """Converts a PIL image into an appropriately transformed tensor."""
        if self.center_crop:
            img = transforms.CenterCrop(size=min(img.size))(img)
        img_t = self._base_img_trans(img)
        # Normalize the tensor to be in the range [-1, 1]
        img_t = (img_t - 0.5) * 2.0
        return img_t

    def __len__(self):
        return len(self.records)

    def __getitem__(self, idx):
        return self.records[idx]
