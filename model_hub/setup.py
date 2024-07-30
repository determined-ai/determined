import os
import setuptools

def version():
    version = os.environ.get("VERSION")

    if version is None:
        try:
            # This feels more disgusting than it is. Numpy does something similar,
            # although they generate a version.py file from their Meson build file
            # that returns a static version string. I'm not thrilled about calling a
            # shell script during Determined's __init__.py (i.e. on import), but I'm
            # running out of ideas to make editable installs work comfortably, and
            # this shouldn't ever run for end users anyway.
            output = subprocess.run(["../version.sh"], capture_output=True)
        except subprocess.CalledProcessError:
            # version.sh failed for whatever reason. Return an unknown version with
            # epoch set to 1 so at least pip dependency resolution should succeed.
            version = "1!0.0.0+unknown"
        else:
            # version.sh succeeded. Collect the output.
            version = output.stdout.decode("utf-8")

    return version

setuptools.setup(
    version=version()
)
