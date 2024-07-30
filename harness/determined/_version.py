import importlib.metadata

import os
import subprocess

def version(name):
    version = None

    try:
        # Attempt to get the version from the package metadata. This should
        # exist if using a built distribution (i.e. for most users), which
        # includes pip editable installations.
        version = importlib.metadata.version(name)
    except importlib.metadata.PackageNotFoundError:
        # In editable mode, or otherwise not running from a distribution. Try
        # calling version.sh next.
        pass
    else:
        return version

    try:
        # This feels more disgusting than it is. Numpy does something similar,
        # although they generate a version.py file from their Meson build file
        # that returns a static version string. I'm not thrilled about calling a
        # shell script during Determined's __init__.py (i.e. on import), but it
        # should only run for things that use the Python source directly, like
        # pytest.
        output = subprocess.run(["./version.sh"], capture_output=True)
    except subprocess.CalledProcessError:
        # version.sh failed for whatever reason. Return an unknown version with
        # epoch set to 1 so at least pip dependency resolution should succeed.
        version = "1!0.0.0+unknown"
    else:
        # version.sh succeeded. Collect the output.
        version = output.stdout.decode("utf-8")

    return version
