import contextlib
import os
import sys
from importlib import machinery
from typing import Iterator, Set, no_type_check


class NoCachePathFinder(machinery.PathFinder):
    """Prevent __pycache__ files from being added to the checkpoint."""

    @no_type_check
    def find_spec(self, name, path=None, target=None):
        spec = super().find_spec(name, path, target)
        if spec is not None and spec.loader is not None:
            old_loader = spec.loader

            class NoCacheLoader(type(old_loader)):  # type: ignore
                def __init__(wrapper) -> None:
                    pass

                def set_data(wrapper, *args, **kwarg):
                    # We don't want to generate __pycache__
                    # directories in the checkpoint dir.
                    raise NotImplementedError()

                def __getattr__(wrapper, *args, **kwargs):
                    return getattr(old_loader, *args, **kwargs)

                def __setattr__(wrapper, *args, **kwargs):
                    return setattr(old_loader, *args, **kwargs)

                def __delattr__(wrapper, *args, **kwargs):
                    return delattr(old_loader, *args, **kwargs)

            spec.loader = NoCacheLoader()

        return spec


def modules_from_dir(path: str) -> Set[str]:
    """List any sys.modules that were imported from a given path."""
    abspath = os.path.abspath(path)
    keys_to_pop = set()
    # sort items before processing, so we always see x before x.y before x.y.z
    for k, v in sorted(sys.modules.items()):
        if v is None:
            # cached miss in sys.modules
            keys_to_pop.add(k)
            continue

        if getattr(v, "__file__", None) is None:
            # non-file module, ignore
            continue

        # Look for dir containing the module.
        dirname, basename = os.path.split(v.__file__)
        if basename == "__init__.py":
            dirname, basename = os.path.split(dirname)

        if dirname == abspath or dirname in keys_to_pop:
            # module was imported from path, or is a submodule of such a module
            keys_to_pop.add(k)
    return keys_to_pop


_in_import_from_path = False


@contextlib.contextmanager
def import_from_path(path: os.PathLike) -> Iterator:
    """
    import_from_path allows you to import from a specific directory and cleans up afterwards.

    Even if you are importing identically-named files, you can import them as separate modules.
    This is intended to help when you have, for example, a current model_def.py, but also import an
    older model_def.py from a checkpoint into the same interpreter, without conflicts (so long as
    you import them as different names, of course).

    Example:

    .. code::

       import model_def as new_model_def

       with det.import_from_path(checkpoint_dir):
           import model_def as old_model_def

           old_model = old_model_def.my_build_model()
           old_model.my_load_weights(checkpoint_dir)

       current_model = new_model_def.my_build_model(
           base_layers=old_model.base_layers
       )

    Without ``import_from_path``, the above code snippet would hit issues where ``model_def`` had
    already been imported so the second ``import`` would have been a noop and both ``new_model_def``
    and ``old_model_def`` would represent the same underlying module.
    """

    global _in_import_from_path
    if _in_import_from_path:
        raise RuntimeError(
            "det.import_from_path unfortunately does not support nesting or calling from multiple "
            "threads."
        )

    fspath = os.fspath(path)
    old_sys_path = sys.path
    popped_modules = {}
    for localpath in ("", os.getcwd()):
        if localpath in sys.path:
            # Remove "" from sys.path.
            sys.path = [p for p in sys.path if p != ""]
            # Remove anything imported from local directory
            for m in modules_from_dir(localpath):
                popped_modules[m] = sys.modules.pop(m)
    # Inject path at front of sys.path.
    sys.path = [os.path.abspath(path)] + sys.path
    # Include a Finder that prevents found files from creating __pycache__ files.
    old_meta_path = sys.meta_path
    sys.meta_path = [NoCachePathFinder()] + sys.meta_path  # type: ignore
    _in_import_from_path = True
    try:
        yield
    finally:
        _in_import_from_path = False
        # Restore meta path.
        sys.meta_path = old_meta_path
        # Remove cached modules loaded from path.
        for m in modules_from_dir(fspath):
            del sys.modules[m]
        # Restore the original sys.path.
        sys.path = old_sys_path
        # Restore local directory modules to sys.modules.
        sys.modules.update(popped_modules)
