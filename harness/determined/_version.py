import importlib.metadata

def version(name):
    version = "1!0.0.0+unknown"

    try:
        # Attempt to get the version from the package metadata. This should
        # exist if using a built distribution (i.e. for most users), which
        # includes pip editable installations.
        version = importlib.metadata.version(name)
    except importlib.metadata.PackageNotFoundError:
        pass

    return version
