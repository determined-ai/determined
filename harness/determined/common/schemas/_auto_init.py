import functools
import inspect
from typing import Any, Callable


def auto_init(old_init: Callable) -> Callable:
    """
    auto_init is a decorator for an __init__ which uses setattr() to set values in __init__ based
    on the signature of the __init__ function.

    Check out this class:

        class Thing:
            a: int
            b: Optional[TestSub] = None

            @auto_init
            def __init__(
                self,
                a: int,
                b: Optional[TestSub] = None,
            ):
                ## This is effectively what happens magically due to @auto_init
                # if a is not None:
                #     self.a = a
                # if b is not None:
                #     self.b = b
                pass

    The simpler strategy would be to skip the annotations at the Class level and just set values
    in __init__.  However, using @auto_init has several benefits:

      - By relying on class annotations for default values, you can always call thing.a, but you
        can also use `"a" in vars(thing)` to know if the value was set explicitly or not.

      - The annotations are easily recognized by type-aware systems for linting or tab-completion.

      - Listing the types in the signature of ``__init__()`` is not actually necessary (it could
        be inferred from the annotations) but for type-awareness and tab-completion systems it is
        necessary.  (side note: you don't need explicit ``__init__`` definitions for ``@dataclass``
        classes when working with mypy, but that's because mypy special-cases them.)

      - Given the previous point, the annotations and the signature can easily be kept in perfect
        sync with each other.  Enforcing the synchronization between the ``__init__`` signature and
        the body of ``__init__`` instead would be difficult with a large body of evolving configs.
    """

    old_sig = inspect.signature(old_init)

    @functools.wraps(old_init)
    def set_all_attrs(self: Any, *args: list, **kwargs: dict) -> None:
        if args:
            raise TypeError(
                f"{type(self).__name__} must be initialized with only keyword arguments"
            )

        try:
            kw = dict(old_sig.bind(self, *args, **kwargs).arguments)
        except TypeError as e:
            # Display the class name in the TypeError.
            raise TypeError(f"{type(self).__name__}: {e}")

        del kw["self"]
        for k, v in kw.items():
            setattr(self, k, v)

        # Always call the old __init__ in case there is anything useful in there.
        old_init(self, **kwargs)

    return set_all_attrs
