import contextlib
import os
import pathlib
import shutil
import subprocess
import sys
import textwrap
from typing import Iterator

import pytest

import determined as det


def test_import_side_effects() -> None:
    # In a separate python process from pytest, import some common parts of
    # determined and ensure that no expensive imports are imported as side effects.
    script = """
        import sys
        import re
        import importlib

        # Make sure this doesn't import azure, boto3/botocore, or google.
        import determined.common.storage

        # Make sure these don't import numpy.
        import determined.tensorboard
        import determined.util

        # Make sure this doesn't import zmq.
        import determined.ipc

        # Make sure that basic operation of the cli is also ok.
        import determined.cli.cli

        if __name__ == "__main__":
            determined.cli.cli.main()

            bad = {
                "^lomond\\.*",
                "^pathspec\\.*",
                "^google\\.*",
                "^boto3\\.*",
                "^botocore\\.*",
                "^azure\\.*",
                "^numpy\\.*",
                "^tensorflow\\.*",
                "^keras\\.*",
                "^pytorch\\.*",
                "zmq\\.*",
            }

            # Detect modules we identified as too expensive...
            found = [m for p in bad for m in sys.modules if re.match(p, m)]

            # But allow namespace modules, which have no __file__ attribute.
            # I haven't actually figured out where these come from, but they are harmless.
            bad = []
            for f in found:
                m = importlib.import_module(f)
                if hasattr(m, "__file__") and m.__file__ is not None:
                    bad.append(f)
            assert not bad, bad
    """
    subprocess.run([sys.executable, "-c", textwrap.dedent(script)], check=True)


def test_import_from_path() -> None:
    @contextlib.contextmanager
    def prepend_sys_path(path: str) -> Iterator:
        old = sys.path
        sys.path = [path] + sys.path
        try:
            yield
        finally:
            sys.path = old

    @contextlib.contextmanager
    def chdir(path: pathlib.Path) -> Iterator:
        old = os.getcwd()
        os.chdir(str(path))
        try:
            yield
        finally:
            os.chdir(old)

    fixture = pathlib.Path(__file__).parent / "fixtures" / "import_from_path"
    # Modify sys.path so that lib1.py and lib2.py are treated as if it were an installed library.
    # Installed libraries should not be affected by the caching behavior
    with prepend_sys_path(str(fixture / "libraries")):
        # Import lib1 before the checkpoints do.
        import lib1

        # Execute from the a/ directory like we were in a normal interactive interpreter.
        with chdir(fixture / "a"), prepend_sys_path(""):
            import model_def as a  # noqa: I2001

            with det.import_from_path(fixture / "b"):
                import model_def as b  # noqa: I2001

                # Nesting is not supported.
                with pytest.raises(RuntimeError, match="does not support nesting"):
                    with det.import_from_path(fixture / "c"):
                        pass

            with det.import_from_path(fixture / "c"):
                import model_def as c  # noqa: I2001

        # Import lib2 after the checkpoints do.
        import lib2

    # Each module should have its own val and data_val.
    assert a.val == "a", a.val
    assert a.data_val == "a", a.data_val
    assert b.val == "b", b.val
    assert b.data_val == "b", b.data_val
    assert c.val == "c", c.val
    assert c.data_val == "c", c.data_val

    # Each module must share the same lib1 and lib2, from outside import_from_path.
    assert id(lib1) == a.lib1_id == b.lib1_id == c.lib1_id
    assert id(lib2) == a.lib2_id == b.lib2_id == c.lib2_id

    # Ensure no cache files got created in the import_from_apaths
    assert not os.path.exists(fixture / "b" / "__pycache__")
    assert not os.path.exists(fixture / "c" / "__pycache__")

    # We will have created some __pycache__ dirs but we don't want to pollute.
    shutil.rmtree(fixture / "a" / "__pycache__")
    shutil.rmtree(fixture / "libraries" / "__pycache__")

    # These modules may leak into other imports in tests downstream
    for mod in ["lib1", "lib2", "data"]:
        sys.modules.pop(mod)
