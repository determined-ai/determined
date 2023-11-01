import logging
import pathlib
import re
from typing import List, Optional

logger = logging.getLogger("determined.tensorboard")

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
    ".trace.json",
]

pytorch_profiler_file_extensions = [
    ".pt.trace.json",
    ".pt.trace.json.gz",
]

pytorch_profiler_file_pattern = re.compile(
    r"""^(.*?) # worker name
        (\.\d+)? # optional timestamp like 1619499959628 used as span name
        \.pt\.trace\.json # the ending suffix
        (?:\.gz)?$""",
    re.X,
)  # optional .gz extension


def find_tb_files(base_dir: pathlib.Path) -> List[pathlib.Path]:
    """
    Recursively searches through base_dir and subdirectories to find files
    needed by Tensorboard, currently matching by filenames and extensions.
    This method is used to sync files generated during training to persistent storage.

    :param base_dir: starting directory path
    :return: list of filepaths within base_dir that are relevant Tensorboard files
    """

    if not base_dir.exists():
        logger.warning(f"{base_dir} directory does not exist.")
        return []

    return [file for filetype in tb_file_types for file in base_dir.rglob(filetype)]


def get_rank_aware_path(path: pathlib.Path, rank: int) -> pathlib.Path:
    """
    Add suffix "#{rank}" to the names of tensorboard
    profiler data files; those names are the host names.
    For example, with rank = 3 "2022_05_13_15_25_41/ip-172-31-8-212.input_pipeline.pb"
    will become "2022_05_13_15_25_41/ip-172-31-8-212#3.input_pipeline.pb"
    """

    pytorch_profiler_extension = get_pytorch_profiler_file_extension(path)
    if pytorch_profiler_extension:
        return _get_rank_aware_path_pytorch_profiler(path, pytorch_profiler_extension, rank)

    for ext in profiler_file_extensions:
        if path.match(f"*{ext}"):
            num_parts = ext.count(".")
            while num_parts > 0:
                path = path.with_suffix("")
                num_parts -= 1
            path = path.with_name(f"{path.name}#{rank}{ext}")
            return path
    return path


def _get_rank_aware_path_pytorch_profiler(
    path: pathlib.Path, pytorch_profiler_extension: str, rank: int
) -> pathlib.Path:
    path_parts = path.parts
    file_name = path_parts[-1]
    match = pytorch_profiler_file_pattern.match(file_name)

    if match:
        match_groups = match.groups()

        worker = match_groups[0]
        worker = f"{worker}#{rank}"

        span = match_groups[1]

        if span:
            file_name = f"{worker}{span}{pytorch_profiler_extension}"
        else:
            file_name = f"{worker}{pytorch_profiler_extension}"

        if len(path_parts) == 1:
            # only file name is passed in
            return pathlib.Path(file_name)
        else:
            # path with directory is passed in
            output_path = pathlib.Path(path_parts[0])
            for i in range(1, len(path_parts) - 1):
                output_path = output_path.joinpath(path_parts[i])
            output_path = output_path.joinpath(file_name)
            return output_path
    else:
        raise Exception(
            (
                f"Path: {path} has pytorch profiler extension {pytorch_profiler_extension}",
                "but no matching file pattern",
            )
        )


def get_pytorch_profiler_file_extension(path: pathlib.Path) -> Optional[str]:
    for ext in pytorch_profiler_file_extensions:
        if path.match(f"*{ext}"):
            return ext
    return None
