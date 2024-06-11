import datetime

import pytest

from tests import api_utils, detproc


@pytest.mark.e2e_cpu
def test_cluster_message_when_admin() -> None:
    admin = api_utils.admin_session()

    # Clear and make sure get throws no errors and returns no cluster message
    detproc.check_call(
        admin,
        ["det", "master", "cluster-message", "clear"],
    )
    res = detproc.check_json(admin, ["det", "master", "cluster-message", "get", "--json"])
    assert res["clusterMessage"] is None, "there should not be a cluster message set after clearing"

    # Python's ISO format isn't RFC 3339, so it doesn't include the Z; add it
    end_time = (datetime.datetime.utcnow() + datetime.timedelta(days=500)).isoformat() + "Z"
    test_message = "test message 1"

    # Set the message
    detproc.check_call(
        admin,
        ["det", "master", "cluster-message", "set", "-m", test_message, "--end", end_time],
    )

    # Make sure master info has an active cluster message
    master_info = detproc.check_json(
        admin,
        ["det", "master", "info", "--json"],
    )
    assert (
        master_info["clusterMessage"] is not None
    ), "active cluster message should be visible in master info"

    # Check the result
    actual_message = detproc.check_json(
        admin,
        ["det", "master", "cluster-message", "get", "--json"],
    )["clusterMessage"]
    assert (
        actual_message["message"] == test_message
    ), "the message returned by the API was not the one expected"
    assert (
        actual_message["endTime"] == end_time
    ), "the end time returned by the API was not what was expected"

    # Test setting one with --duration
    test_message = "test message 2"
    start_time = "2035-01-01T00:00:00Z"
    duration = "22h"
    expected_end = "2035-01-01T22:00:00Z"
    detproc.check_call(
        admin,
        [
            "det",
            "master",
            "cluster-message",
            "set",
            "-m",
            test_message,
            "--start",
            start_time,
            "--duration",
            duration,
        ],
    )

    # Make sure master info has no active cluster message since it's scheduled in the future
    master_info = detproc.check_json(
        admin,
        ["det", "master", "info", "--json"],
    )
    assert (
        master_info["clusterMessage"] is None
    ), "cluster message scheduled in the future should not be visible in master info"

    # Make sure cluster message *is* present in result of cluster-message get
    msg = detproc.check_json(admin, ["det", "master", "cluster-message", "get", "--json"])[
        "clusterMessage"
    ]
    assert (
        msg["message"] == test_message
    ), "cluster message returned by the API was not the one expected"
    assert (
        msg["endTime"] == expected_end
    ), "cluster message end time returned by the API was not the one expected"

    # Clear and make sure the cluster message is unset
    detproc.check_call(
        admin,
        ["det", "master", "cluster-message", "clear"],
    )
    resp = detproc.check_json(admin, ["det", "master", "cluster-message", "get", "--json"])
    assert (
        resp["clusterMessage"] is None
    ), "there should not be a cluster message set after clearing"
    master_info = detproc.check_json(
        admin,
        ["det", "master", "info", "--json"],
    )
    assert (
        master_info["clusterMessage"] is None
    ), "cluster message should not be visible in master info after clearing"


@pytest.mark.e2e_cpu
def test_cluster_message_requires_admin() -> None:
    user = api_utils.user_session()
    admin = api_utils.admin_session()

    # Stuff that should fail when not admin
    proc = detproc.run(user, ["det", "master", "cluster-message", "get"])
    assert proc.returncode != 0, "cluster message get should have failed when not admin"
    proc = detproc.run(user, ["det", "master", "cluster-message", "set", "-m", "foobarbaz"])
    assert proc.returncode != 0, "cluster message set should have failed when not admin"
    proc = detproc.run(user, ["det", "master", "cluster-message", "clear"])
    assert proc.returncode != 0, "cluster message clear should have failed when not admin"

    # Actually set a cluster message as admin
    expected_message = "foobarbaz"
    detproc.check_call(admin, ["det", "master", "cluster-message", "set", "-m", expected_message])

    # Verify we see the correct message as a normal user
    master_info = detproc.check_json(user, ["det", "master", "info", "--json"])
    assert (
        master_info["clusterMessage"] is not None
    ), "active cluster messages should be visible to non-admins"

    # Clean up after ourselves
    detproc.run(admin, ["det", "master", "cluster-message", "clear"])
