import pathlib
import shutil
import tempfile
from types import TracebackType  # noqa:I2041
from typing import ContextManager, Dict, Optional, Type, Union


class FileTree(ContextManager[pathlib.Path]):
    """
    FileTree creates a set of files with their contents in their subdirectories
    and cleans them up later.
    """

    def __init__(self, tmp_path: pathlib.Path, files: Dict[Union[pathlib.Path, str], str]) -> None:
        """
        Creates a file tree in tempdir with the given filenames and contents.
        """
        self.tmp_path = tmp_path
        self.files = {pathlib.Path(k): v for k, v in files.items()}
        self.dir = None  # type: Optional[pathlib.Path]

    def __enter__(self) -> pathlib.Path:
        """Creates FileTree and returns the root directory of the FileTree."""
        self.dir = pathlib.Path(tempfile.mkdtemp(dir=str(self.tmp_path)))
        try:
            for name, contents in self.files.items():
                p = self.dir.joinpath(name)
                p.parent.mkdir(parents=True, exist_ok=True)
                p.write_text(contents)
        except OSError:
            shutil.rmtree(str(self.dir), ignore_errors=True)

        return self.dir

    def __exit__(
        self,
        exc_type: Optional[Type[BaseException]],
        exc_value: Optional[BaseException],
        traceback: Optional[TracebackType],
    ) -> None:
        shutil.rmtree(str(self.dir), ignore_errors=True)
