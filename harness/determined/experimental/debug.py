"""
Useful tools for debugging.  This will hopefully be the basis of a public-facing debugging toolkit
for model development, but right now it is undocumented and only used internally.

.. warning::
   The code in this module should be considered totally unstable and may change or be removed at
   any time.
"""

import contextlib
import faulthandler
from typing import Generator


@contextlib.contextmanager
def stack_trace_thread(stack_trace_period_sec: float) -> Generator:
    """
    If enabled, emit stack traces periodically.  This is particularly useful when your model is
    hanging, because the periodic stack traces will show where the hang is occuring.
    """
    if stack_trace_period_sec <= 0.0:
        yield
        return

    faulthandler.dump_traceback_later(stack_trace_period_sec, repeat=True)
    try:
        yield
    finally:
        faulthandler.cancel_dump_traceback_later()
