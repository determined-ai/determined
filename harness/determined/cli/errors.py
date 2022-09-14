class EnterpriseOnlyError(Exception):
    """Exception indicating the master may be missing an EE-only feature."""

    pass


class FeatureFlagDisabled(Exception):
    """
    Exception indicating that there is a currently disabled feature flag
    that is required to use a feature
    """

    pass
