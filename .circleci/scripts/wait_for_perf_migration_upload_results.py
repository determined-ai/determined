import datetime
import os
import pathlib
import re
import subprocess
import time

import psycopg2
import requests
from psycopg2 import extensions, sql

from determined.common import api, util
from determined.common.api import authentication, bindings, certs


def _wait_for_master() -> None:
    print("Checking for master at 127.0.0.1")
    # 2 hours is the most a migration can take, with this setup.
    # If a migration takes longer than that we have hit an issue a customer will likely hit too.
    for i in range(2 * 60 * 60):
        try:
            r = api.get("127.0.0.1", "info", authenticated=False)
            if r.status_code == requests.codes.ok:
                return
        except api.errors.MasterNotFoundException:
            pass
        if i % 60 == 0:
            print("Waiting for master to be available...")
        time.sleep(1)
    raise ConnectionError("Timed out connecting to Master")


def _upload_migration_length(conn: extensions.connection) -> None:
    authentication.cli_auth = authentication.Authentication(
        "http://127.0.0.1:8080",
        requested_user="admin",
        password="",
    )
    sess = api.Session("http://127.0.0.1:8080", "admin", authentication.cli_auth, certs.cli_cert)

    migration_start_log = None
    migration_end_log = None
    no_migration_to_apply_log = None
    for log in bindings.get_MasterLogs(sess):
        if "running DB migrations from" in log.logEntry.message:
            migration_start_log = log.logEntry
            continue

        match = re.search(r"migrated from (\d+) to (\d+)", log.logEntry.message)
        if match:
            from_version = int(match.group(1))
            to_version = int(match.group(2))
            migration_end_log = log.logEntry
            continue

        if "no migrations to apply" in log.logEntry.message:
            no_migration_to_apply_log = log.logEntry

    assert migration_start_log is not None
    if no_migration_to_apply_log is not None:
        print(
            "got no migration to apply message (nothing to record) "
            + f"'{no_migration_to_apply_log.message}'"
        )

        indicate_file = "/tmp/no-migrations-needed"
        print(f"creating file at {indicate_file} to indicate this")
        pathlib.Path(indicate_file).touch()

        assert migration_end_log is None
        return

    assert migration_end_log is not None
    print(
        f"migration start message '{migration_start_log.message}' "
        + f"at {migration_start_log.timestamp}"
    )
    print(f"migration end message '{migration_end_log.message}' at {migration_end_log.timestamp}")

    start_ts = util.parse_protobuf_timestamp(migration_start_log.timestamp)
    end_ts = util.parse_protobuf_timestamp(migration_end_log.timestamp)
    duration = (end_ts - start_ts) / datetime.timedelta(microseconds=1)
    print(f"migrating {from_version} to {to_version} took {duration}ms")

    try:
        commit = subprocess.check_output(
            ["git", "log", "-1", "--pretty=format:%H"], universal_newlines=True
        ).strip()
    except subprocess.CalledProcessError:
        commit = "unknown"

    run_sql_query = sql.SQL(
        """
        INSERT INTO migration_runs (commit, branch, duration_ms, from_version, to_version)
        VALUES ({commit}, {branch}, {duration_ms}, {from_version}, {to_version})
        RETURNING id;
    """
    ).format(
        commit=sql.Literal(commit),
        branch=sql.Literal(os.environ.get("CIRCLE_BRANCH", "unknown")),
        duration_ms=sql.Literal(duration),
        from_version=sql.Literal(from_version),
        to_version=sql.Literal(to_version),
    )

    cursor = conn.cursor()
    cursor.execute(run_sql_query)
    conn.commit()


if __name__ == "__main__":
    db_params = {
        "host": os.environ["PERF_RESULT_DB_HOST"],
        "user": os.environ["PERF_RESULT_DB_USER"],
        "password": os.environ["PERF_RESULT_DB_PASS"],
        "dbname": "postgres",
    }
    connection = psycopg2.connect(**db_params)

    _wait_for_master()

    _upload_migration_length(connection)
