from typing import Any, Dict, Tuple, Union

from tests.experiment import utils


def test_assert_events_match() -> None:
    """
    Make sure our test utility actually works, since it is the basis for
    ensuring that our callback-based 3rd-party integrations works.
    """

    def expect_success(
        events: utils.Events, *patterns: Union[str, Tuple[str, str]]
    ) -> Dict[str, Any]:
        try:
            return utils.assert_events_match(events, *patterns)
        except AssertionError:
            raise AssertionError(f"expected success: {patterns}")

    def expect_failure(events: utils.Events, *patterns: str) -> None:
        try:
            utils.assert_events_match(events, *patterns)
        except AssertionError:
            pass
        else:
            raise AssertionError(f"expected failure: {patterns}")

    events = utils.Events([("1", None), ("2", 2), ("3", None)])

    expect_success(events, "1")
    expect_success(events, "2")
    expect_success(events, "3")
    expect_success(events, "1", "2", "3")
    expect_success(events, "1", "!4")
    expect_success(events, "!0", "2")
    expect_success(events, "!2", "1", "2")
    expect_success(events, "[0-3]", "[0-3]", "[0-3]")
    # Make sure a positive match takes precedence over a negative match.
    expect_success(events, "![3-9]", "3")

    expect_failure(events, "1", "3", "4")
    expect_failure(events, "1", "!2")
    expect_failure(events, "1", "!2", "3")
    expect_failure(events, "!1", "2")

    # Make sure we capture the data for events like we expect.
    assert expect_success(events, "1", ("2", "two"), "3") == {"two": 2}
