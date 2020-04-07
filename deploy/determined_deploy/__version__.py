def _get_version() -> str:
    import pkg_resources

    return pkg_resources.get_distribution("determined_deploy").version


__version__ = _get_version()
