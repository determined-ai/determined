import abc
import datetime
from typing import Any, Callable, Dict, Generator, List


class Fetcher(metaclass=abc.ABCMeta):
    """Abstract base class for TensorBoard fetchers.

    Syncs TensorBoard files from remote file blob stores.
    """

    storage_paths: List[str]
    _file_records: Dict[str, datetime.datetime] = {}

    @abc.abstractmethod
    def __init__(self, storage_config: Dict[str, Any], storage_paths: List[str], local_dir: str):
        pass

    @abc.abstractmethod
    def _list(self, storage_path: str) -> Generator[str, None, None]:
        """Iterates over the remote directory storage_path and yields any file that is new or
        has an updated timestamp from when it was last fetched.

        Arguments:
            storage_path (str): Path at a remote location to iterate over
            new_file_callback (Callable, optional): Callback function that
                is fired each time a new file is fetched
        """
        pass

    @abc.abstractmethod
    def _fetch(self, filepath: str, new_file_callback: Callable) -> None:
        """Performs actual file fetch from the remote filepath to the internal local_dir

        Arguments:
            filepath (str): Filepath as a string of the file's remote location
            new_file_callback (Callable, optional): Callback function that
                is fired each time a new file is fetched
        """
        pass

    def list_all_generator(self) -> Generator[str, None, None]:
        """Iterates over all files that need to be fetched"""
        for storage_path in self.storage_paths:
            for filepath in self._list(storage_path):
                yield filepath
