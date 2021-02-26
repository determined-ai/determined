import contextlib
import functools
from typing import Any, Callable, Iterator


@contextlib.contextmanager
def monkey_patch(obj: Any, name: str, func: Callable[..., Any]) -> Iterator:
    orig = getattr(obj, name)

    @functools.wraps(orig)
    def new(*args, **kwargs):  # type: ignore
        return func(orig, *args, **kwargs)

    setattr(obj, name, new)

    try:
        yield new
    finally:
        setattr(obj, name, orig)


def monkey_patch_decorator(obj: Any, attr: str) -> Callable:
    orig = getattr(obj, attr)

    def wrap(f):  # type: ignore
        @functools.wraps(f)
        def inner(*args, **kwargs):  # type: ignore
            return f(orig, *args, **kwargs)

        setattr(obj, attr, inner)

    return wrap
