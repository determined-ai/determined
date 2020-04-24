import logging
import sys


def _set_logger(debug_enabled: bool) -> None:
    root = logging.getLogger()
    root.setLevel(logging.DEBUG if debug_enabled else logging.INFO)
    handler = logging.StreamHandler(sys.stdout)
    handler.setLevel(logging.DEBUG)
    formatter = logging.Formatter("%(levelname)s: %(message)s")
    handler.setFormatter(formatter)
    root.addHandler(handler)
