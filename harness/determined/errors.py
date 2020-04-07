from typing import Any, Dict


class InternalException(Exception):
    def __init__(self, message: str) -> None:
        self.message = (
            "Internal error: {}. Please reach out to the "
            "Determined AI team for help.".format(message)
        )

    def __str__(self) -> str:
        return self.message


class InvalidExperimentException(BaseException):
    """
    InvalidExperimentException is used if the model of an experiment is invalid.
    """

    def __init__(self, message: str) -> None:
        super().__init__(message)


class InvalidConfigurationException(InvalidExperimentException):
    """
    InvalidConfigurationException is used if the configuration of an experiment is invalid.
    """

    def __init__(self, config: Dict[str, Any], message: str) -> None:
        super().__init__("{}: {}".format(message, config))


class WorkerError(Exception):
    """
    WorkerError indicates that a worker process failed but we do not know why.
    """

    pass


class WorkerFinishedGracefully(Exception):
    pass


class SkipWorkloadException(Exception):
    """
    Exception that a Trial can raise in order to indicate to the harness that
    this phase can be skipped.
    """

    pass
