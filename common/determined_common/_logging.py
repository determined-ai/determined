import logging
import sys


def set_logger(debug_enabled: bool) -> None:
    root = logging.getLogger()
    root.setLevel(logging.DEBUG if debug_enabled else logging.INFO)

    for hdlr in root.handlers:
        root.removeHandler(hdlr)

    handler = logging.StreamHandler(sys.stdout)
    handler.setLevel(logging.DEBUG if debug_enabled else logging.INFO)
    formatter = logging.Formatter("%(levelname)s: %(message)s")
    handler.setFormatter(formatter)
    root.addHandler(handler)
