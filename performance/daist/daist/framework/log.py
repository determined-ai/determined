import logging
import sys
import time
from pathlib import Path

from ..utils import timestamp
from .paths import PkgPath

FILENAME = 'test_run.log'


class StdoutFilter(logging.Filter):
    def __init__(self, level):
        super().__init__()
        self._level = level

    def filter(self, record: logging.LogRecord) -> bool:
        return record.levelno >= self._level


def start(path: Path, file_level: int, stdout_level: int):
    logger = logging.getLogger(PkgPath.PATH.name)
    logger.setLevel(logging.DEBUG)

    fmt = logging.Formatter(f'|%(asctime)s{timestamp.UTC_Z} '
                            f'%(name)s '
                            f'%(levelname)s|\n'
                            f'%(message)s',
                            timestamp.BASE_FMT)
    fmt.converter = time.gmtime

    file_hdlr = logging.FileHandler(path)
    file_hdlr.setLevel(file_level)
    file_hdlr.setFormatter(fmt)

    stdout_hdlr = logging.StreamHandler(stream=sys.stdout)
    stdout_hdlr.setLevel(stdout_level)
    stdout_hdlr.setFormatter(fmt)
    stdout_hdlr.addFilter(StdoutFilter(stdout_level))

    stderr_hdlr = logging.StreamHandler(stream=sys.stderr)
    stderr_hdlr.setLevel(logging.ERROR)
    stderr_hdlr.setFormatter(fmt)

    logger.addHandler(file_hdlr)
    logger.addHandler(stdout_hdlr)
    logger.addHandler(stderr_hdlr)
