import setuptools

def readme():
    with open("../README.md", "r") as fd:
        return fd.read()

setuptools.setup(
    # We can't seem to use pyproject.toml to include the Determined README
    # relative to pyproject.toml. But that's okay, because we can still keep
    # setup.py for this.
    long_description=readme(),
    long_description_content_type="text/markdown",
)
