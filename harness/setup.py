import os
import subprocess

import setuptools


def readme() -> str:
    with open("../README.md", "r") as fd:
        return fd.read()


def version() -> str:
    def get_version_from_sh():
        try:
            # This feels more disgusting than it is. Numpy does something similar,
            # although they generate a version.py file from their Meson build file
            # that returns a static version string. I'm not thrilled about calling a
            # shell script during Determined's __init__.py (i.e. on import), but I'm
            # running out of ideas to make editable installs work comfortably, and
            # this shouldn't ever run for end users anyway.
            output = subprocess.run(["../version.sh"], capture_output=True, shell=True)
        except subprocess.CalledProcessError:
            # version.sh failed for whatever reason. Return an unknown version with
            # epoch set to 1 so at least pip dependency resolution should succeed.
            return "1!0.0.0+unknown"
        else:
            # version.sh succeeded. Collect the output.
            return output.stdout.decode("utf-8")

    return os.environ.get("VERSION", get_version_from_sh())


setuptools.setup(
    # We can't seem to use pyproject.toml to include the Determined README
    # relative to pyproject.toml. But that's okay, because we can still keep
    # setup.py for this.
    long_description=readme(),
    long_description_content_type="text/markdown",
    version=version(),
)
