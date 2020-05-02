import logging
from typing import Any, Callable

import tensorpack as tp
import zmq
from tensorflow.train import SessionManager
from tensorpack.callbacks.steps import MaintainStepCounter

from determined import monkey_patch

logging.debug("Applying tensorpack patches.")


# TODO(DET-2708): remove zmq patching.
@monkey_patch.monkey_patch_decorator(zmq.Socket, "__del__")
def ignore_close_message(orig_func: Callable, *args: Any, **kwargs: Any) -> Any:
    """
    ZMQ throws unnecessary TypeErrors when closing sockets. We decorate it
    to create a more beautiful experience for our users.
    """
    try:
        ret = orig_func(*args, **kwargs)
    except TypeError:
        ret = None
    return ret


@monkey_patch.monkey_patch_decorator(SessionManager, "__init__")
def set_default_recovery(orig_func: Callable, *args: Any, **kwargs: Any) -> Any:
    """
    Tensorpack instantiates a SessionManager for each worker. If a non-chief worker starts before
    the chief, it waits a certain amount of time before trying to contact the chief again. That time
    is 30 seconds by default, which is quite long; we override it to 1 second here.
    """
    kwargs.setdefault("recovery_wait_secs", 1)
    return orig_func(*args, **kwargs)


@monkey_patch.monkey_patch_decorator(tp.Trainer, "register_callback")
def replace_old_step_callback(orig_func: Callable, self: Any, cb: Any) -> Any:
    """
    Avoid the use of the MaintainStepCounter callback for distributed training due
    to issues with restoring the correct step value, so we maintain the step
    value ourselves.
    """
    if isinstance(cb, MaintainStepCounter):
        return
    return orig_func(self, cb)
