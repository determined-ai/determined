import subprocess
import sys
import textwrap


def test_imports() -> None:
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
