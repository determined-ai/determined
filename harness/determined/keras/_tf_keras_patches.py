import logging

import zmq

from determined import monkey_patch

logging.debug("Applying tf.keras patches.")


@monkey_patch.monkey_patch_decorator(zmq.Socket, "__del__")
def ignore_close_message(orig_func, *args, **kwargs):  # type: ignore
    """
    ZMQ throws unnecessary TypeErrors when closing sockets. We decorate it
    to create a more beautiful experience for our users.
    """
    try:
        ret = orig_func(*args, **kwargs)
    except TypeError:
        ret = None
    return ret
