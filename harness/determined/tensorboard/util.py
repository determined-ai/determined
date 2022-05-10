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

profiler_file_extensions = [
    ".input_pipeline.pb",
    ".memory_profile.json.gz",
    ".tensorflow_stats.pb",
    ".xplane.pb",
    ".kernel_stats.pb",
    ".overview_page.pb",
    ".trace.json.gz",
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


def get_rank_aware_path(p: pathlib.Path, rank: int) -> pathlib.Path:
    for ext in profiler_file_extensions:
        if p.match(f"*{ext}"):
            print(f"matching *{ext}")
            num_parts = ext.count(".")
            while num_parts > 0:
                p = p.with_suffix("")
                num_parts -= 1
            p = p.with_name(f"{p.name}#{rank}")
            p = p.with_suffix(ext)
            return p
    return p
