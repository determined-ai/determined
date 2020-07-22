import pytest


@pytest.mark.distributed  # type: ignore
def test_nothing() -> None:
    """
    This is to work around that pytest returns a non-zero exit code if there exist
    no distributed tests.
    """
    pass
