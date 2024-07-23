import os
import setuptools

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
    version=version()
)
