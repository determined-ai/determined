class FeatureFlagDisabled(Exception):
    """
    Exception indicating that there is a currently disabled feature flag
    that is required to use a feature
    """

    pass


class CliError(Exception):
    """
    Base class for all CLI errors.
    """

    name: str

    def __init__(self, message: str, exit_code: int = 1) -> None:
        """
        Args:
        - e_stack: The exception that triggered this error.
        - exit_code: The exit code to use when exiting the CLI.
        """
        super().__init__(message)
        self.name = "Error"
        self.exit_code = exit_code
        self.message = message
