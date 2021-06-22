import logging
import pathlib
from typing import List

tb_file_types = [
    "*tfevents*",
    "*.trace.json.gz",
    "*.trace.json",
    "*.memory_profile.json.gz",
    "*.pb",
]


def find_tb_files(base_dir: pathlib.Path) -> List[pathlib.Path]:
    """
    Recursively searches through base_dir and subdirectories to find files
    needed by Tensorboard, currently matching by filenames and extensions.
    This method is used to sync files generated during training to persistent storage.

    :param base_dir: starting directory path
    :return: list of filepaths within base_dir that are relevant Tensorboard files
    """

    if not base_dir.exists():
        logging.warning(f"{base_dir} directory does not exist.")
        return []

    return [file for filetype in tb_file_types for file in base_dir.rglob(filetype)]
