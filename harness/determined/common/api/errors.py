from typing import Any

import requests


class BadRequestException(Exception):
    def __init__(self, message: str, *args: Any) -> None:
        super().__init__(message, *args)
        self.message = message

    def __str__(self) -> str:
        return self.message


class BadResponseException(Exception):
    def __init__(self, message: str, *args: Any) -> None:
        super().__init__(message, *args)
        self.message = message

    def __str__(self) -> str:
        return self.message


class MasterNotFoundException(BadRequestException):
    pass


class APIException(BadRequestException):
    """Raised when an API operation has failed.

    APIException is a catchall for errors passed on from the REST API server that aren't
    otherwise classified as other types of exceptions.

    Attributes:
        response_error: A dict parsed from the Response body containing structured error information
        status_code: The HTTP status code of the response
        message: A string containing a human-friendly error message. Inherited from
                 BadRequestException
    """

    def __init__(self, response: requests.Response, *args: Any) -> None:
        """Initialization from the Response that it was raised from.

        Args:
            response: A requests.Response with a body that looks like:
                {
                    error: {
                        code: <int>,
                        reason: <error type>,
                        error: <detailed error message>
                    }
                }
        """
        try:
            self.response_error = response.json()["error"]
            m = self.response_error["error"]
        except (ValueError, KeyError):
            self.response_error = None
            m = response.text
        super().__init__(m, response, *args)
        self.status_code = response.status_code


class NotFoundException(APIException):
    """The internal API's analog to a 404 Not Found HTTP status code."""

    def __init__(self, error_message: str) -> None:
        self.message = error_message
        self.status_code = 404


class ForbiddenException(BadRequestException):
    """The internal API's analog to a 403 Forbidden HTTP status code."""

    def __init__(self, message: str = ""):
        err_message = f"Forbidden({message})"
        if not (message == "invalid credentials" or message == "user not found"):
            err_message += ": Please contact your administrator in order to access this resource."

        super().__init__(message=err_message)


class UnauthenticatedException(BadRequestException):
    """The internal API's analog to a 401 Unauthorized HTTP status code."""

    def __init__(
        self,
        message: str = (
            "Unauthenticated: Please use 'det user login <username>' for password login, or"
            " for Enterprise users logging in with an SSO provider,"
            " use 'det auth login --provider=<provider>'."
        ),
    ) -> None:
        super().__init__(message=message)


class CorruptTokenCacheException(Exception):
    def __init__(self) -> None:
        super().__init__(
            "Attempted to read a corrupted token cache.  The store has been deleted; "
            "please try again.\n"
        )


class CorruptCertificateCacheException(Exception):
    def __init__(self) -> None:
        super().__init__(
            "Attempted to read a corrupted certificate cache.  The store has been deleted; "
            "please try again.\n"
        )


class EmptyResultException(BadResponseException):
    def __init__(self, message: str) -> None:
        super().__init__(message)


class DeleteFailedException(APIException):
    def __init__(self, error_message: str) -> None:
        self.message = error_message
        self.status_code = 200


class InvalidCredentialsException(UnauthenticatedException):
    def __init__(self) -> None:
        super().__init__(message="Invalid username/password combination. Please try again.")
