import contextlib
import os
import pathlib
import shutil
import tempfile
from typing import Iterator


@contextlib.contextmanager
def use_test_config_dir() -> Iterator[pathlib.Path]:
    config_dir = pathlib.Path(tempfile.mkdtemp(prefix="determined-config"))
    try:
        os.environ["DET_DEBUG_CONFIG_PATH"] = str(config_dir)
        yield config_dir
    finally:
        shutil.rmtree(config_dir)
