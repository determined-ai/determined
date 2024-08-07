from typing import Callable


class StreamToLogger:
    def __init__(self, log_func: Callable[[str], None]):
        self._log_func = log_func

    def write(self, msg: str):
        for line in msg.rstrip().splitlines():
            self._log_func(line.rstrip())

    def flush(self):
        pass
