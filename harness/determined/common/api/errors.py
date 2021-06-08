import requests


class BadRequestException(Exception):
    def __init__(self, message: str) -> None:
        self.message = message

    def __str__(self) -> str:
        return self.message


class BadResponseException(Exception):
    def __init__(self, message: str) -> None:
        self.message = message

    def __str__(self) -> str:
        return self.message


class MasterNotFoundException(BadRequestException):
    def __init__(self, message: str) -> None:
        super().__init__(message)


class APIException(BadRequestException):
    """
    Raised when an API operation has failed. The status code is provided in
    the failure.
    """

    def __init__(self, response: requests.Response) -> None:
        try:
            m = response.json()["message"]
        except (ValueError, KeyError):
            m = response.text
        super().__init__(m)
        self.status_code = response.status_code


class NotFoundException(APIException):
    pass


class UnauthenticatedException(BadRequestException):
    def __init__(self, username: str):
        super().__init__(
            message="Unauthenticated: Please use 'det user login <username>' for password login, or"
            " for Enterprise users logging in with an SSO provider,"
            " use 'det auth login --provider=<provider>'."
        )
        self.username = username


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
