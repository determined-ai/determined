import logging
import sys


def set_logger(debug_enabled: bool) -> None:
    root = logging.getLogger()
    root.setLevel(logging.DEBUG if debug_enabled else logging.INFO)

    for hdlr in root.handlers:
        root.removeHandler(hdlr)

    handler = logging.StreamHandler(sys.stdout)
    handler.setLevel(logging.DEBUG if debug_enabled else logging.INFO)
    # If this format is changed, we must update fluentbit, which attempts to parse these logs.
    formatter = logging.Formatter("%(levelname)s: [%(process)s] %(name)s: %(message)s")
    handler.setFormatter(formatter)
    root.addHandler(handler)
