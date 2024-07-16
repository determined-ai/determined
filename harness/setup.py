import os
import setuptools

def readme():
    with open("../README.md", "r") as fd:
        return fd.read()

def version():
    # VERSION should be set by the harness Makefile, obtained from running
    # version.sh at the repository root. This means either the build has to be
    # done via make, or VERSION has to be explicitly set to run python -m build
    # directly. Given the intended use cases, though, this should be fine. It's
    # also more consistent, as it doesn't rely on duplicating version-finding
    # behavior in Python, which has the possibility of drifting if we update one
    # version discovery script but not the other.
    version = os.environ.get("VERSION")
    if version is None:
        raise Exception("VERSION environment variable must be set.")

    return version

setuptools.setup(
    # We can't seem to use pyproject.toml to include the Determined README
    # relative to pyproject.toml. But that's okay, because we can still keep
    # setup.py for this.
    long_description=readme(),
    long_description_content_type="text/markdown",
    version=version()
)
