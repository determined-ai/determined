from typing import Any, Dict, Type


class EnterpriseOnlyError(Exception):
    """Exception indicating the master may be missing an EE-only feature."""

    pass


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
    InvalidExperimentException is used if an experiment is invalid.
    """


class InvalidDataTypeException(InvalidExperimentException):
    """
    InvalidDataType is used if the data type of an experiment is invalid.
    """

    def __init__(self, typ: Type, message: str) -> None:
        super().__init__(f"Invalid data type ({typ.__name__}): {message}.")


class InvalidConfigurationException(InvalidExperimentException):
    """
    InvalidConfigurationException is used if the configuration of an experiment is invalid.
    """

    def __init__(self, config: Dict[str, Any], message: str) -> None:
        super().__init__(f"Invalid configuration ({config}): {message}.")


class InvalidCheckpointException(Exception):
    """
    InvalidCheckpointException is used if a checkpoint is invalid.
    """

    def __init__(self) -> None:
        super().__init__("Invalid checkpoint.")


class StopLoadingImplementation(Exception):
    """
    Exception that intercepts loading the user code.
    """

    pass


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


class InvalidModelException(InvalidExperimentException):
    """
    InvalidModelException indicates that the model is inavlid or is (partially)
    unsupported
    """


class CheckpointNotFound(Exception):
    """
    CheckpointNotFoundException indicates a checkpoint could not be found in checkpoint storage.
    """


class CheckpointStateException(Exception):
    """CheckpointStateException indicates a checkpoint is in an inappropriate state."""

    pass


class NoDirectStorageAccess(Exception):
    """Direct checkpoint storage access unavailable, e.g., no credentials or permissions."""

    pass


class ProxiedDownloadFailed(Exception):
    """Proxied checkpoint download through master failed"""

    pass


class MultipleDownloadsFailed(Exception):
    """Multiple checkpoint download methods failed"""

    pass
