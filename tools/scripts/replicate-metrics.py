#!/usr/bin/python3

"""
connect to Determined's db and replicate various metrics for performance testing purposes.
"""

import psycopg  # pip install "psycopg[binary]"
import os
import contextlib
from typing import Generator, Set, Union
from psycopg.abc import Query
import time

DB_NAME = os.environ.get("DET_DB_NAME", "determined")
DB_USERNAME = os.environ.get("DET_DB_USERNAME", "postgres")
DB_PASSWORD = os.environ.get("DET_DB_PASSWORD", "postgres")
DB_HOST = os.environ.get("DET_DB_HOST", "localhost")
DB_PORT = os.environ.get("DET_DB_PORT", "5432")


"""
- connect to db
- create new trial records
    - select completed trials
        - maybe select all trial states
    - maybe zeroout checkpoint size, count, restarts, task and request ids.
    - vary state between completed and errored
    - add synth tag
- replicate steps and validation records
    - maybe vary total_batches? offset
    - update trial_id and trial_run_id
- bulk update trial ids.
- check that related endpoints work


TODO:
- we'd probably want to only pick trials that belong to multi trial experiments
"""

# a class extending psycopg.Cursor that adds logging around each query execute.
class LoggingCursor(psycopg.Cursor):
    def execute(self, query: Union[Query, str], *args, **kwargs) -> None:
        print(
            f"""====QUERY START====
{query.strip() if isinstance(query, str) else query}
====QUERY END===="""
        )
        start = time.time()
        super().execute(query, *args, **kwargs)  # type: ignore
        end = time.time()
        print("query took (ms):", (end - start) * 1000)


@contextlib.contextmanager
def db_cursor() -> Generator[psycopg.Cursor, None, None]:
    conn = psycopg.connect(
        dbname=DB_NAME,
        user=DB_USERNAME,
        password=DB_PASSWORD,
        host=DB_HOST,
        port=DB_PORT,
    )
    conn.cursor_factory = LoggingCursor
    yield conn.cursor()
    conn.close()


def get_table_col_names(table: str) -> Set[str]:
    with db_cursor() as cur:
        cur.execute(
            """
            SELECT column_name FROM information_schema.columns WHERE table_name = %s
            """,
            (table,),
        )
        rows = cur.fetchall()
        return {row[0] for row in rows}


def replicate_rows(table: str, skip_cols: Set[str], multiplier=1, suffix="") -> None:
    cols = get_table_col_names(table)
    cols = ", ".join(cols - skip_cols)

    with db_cursor() as cur:
        query = f"""
INSERT INTO {table}( {cols} )
SELECT {cols} FROM {table}
CROSS JOIN generate_series(1, {multiplier}) AS g
{suffix};
        """
        cur.execute(query)  # type: ignore
        print("added rows:", cur.rowcount)
        cur.execute("COMMIT")


def copy_trial(trial_id: int) -> None:
    """
    copy a single trial with associated metrics using CTE.
    """
    trial_cols = get_table_col_names("trials") - {"id"}
    trial_cols_str = ", ".join(trial_cols)
    steps_cols = get_table_col_names("raw_steps") - {"id", "trial_id"}
    steps_cols_str = ", ".join(steps_cols)
    prefixed_steps_cols = ", ".join([f"rs.{col}" for col in steps_cols])
    validations_cols = get_table_col_names("raw_validations") - {"id", "trial_id"}
    validations_cols_str = ", ".join(validations_cols)
    prefixed_validations_cols = ", ".join([f"rv.{col}" for col in validations_cols])

    with db_cursor() as cur:
        query = f"""
WITH replicated_trials AS (
INSERT INTO trials ({trial_cols_str})
SELECT {trial_cols_str}
FROM trials
WHERE id = %s
RETURNING id
), replicated_steps AS (
INSERT INTO raw_steps (trial_id, {steps_cols_str})
SELECT rt.id, {prefixed_steps_cols}
FROM replicated_trials rt
JOIN raw_steps rs ON rs.trial_id = %s
RETURNING trial_id, id AS new_step_id
)
INSERT INTO raw_validations (trial_id, {validations_cols_str})
SELECT rt.id, {prefixed_validations_cols}
FROM replicated_trials rt
JOIN raw_validations rv ON rv.trial_id = %s;
        """
        cur.execute(query, (trial_id, trial_id, trial_id))  # type: ignore
        cur.execute("COMMIT")


def copy_trials(multiplier=1, suffix="") -> dict:
    table = "trials"
    assert multiplier == 1  # disable this for now
    cols = get_table_col_names(table) - {"id"}
    cols_str = ", ".join(cols)

    with db_cursor() as cur:
        query = f"""
-- modify the target table to add a new col called og_id
ALTER TABLE {table} ADD COLUMN og_id int;
"""
        cur.execute(query)  # type: ignore

        query = f"""
-- insert the replicated rows populating the og_id column with the original id
INSERT INTO {table}( {cols_str}, og_id )
SELECT {cols_str}, id
FROM {table}
-- CROSS JOIN generate_series(1, {multiplier}) AS g
{suffix};
"""
        cur.execute(query)  # type: ignore
        replicated_trials = cur.rowcount

        steps_cols = get_table_col_names("raw_steps") - {"id"} - {"trial_id"}
        steps_cols_str = ", ".join(steps_cols)
        prefixed_steps_cols = ", ".join([f"rs.{col}" for col in steps_cols])
        # replicate raw_steps and update trial_id
        query = f"""

-- replicate raw_steps and keep the new step ids
INSERT INTO raw_steps( {steps_cols_str}, trial_id )
SELECT {prefixed_steps_cols}, trials.id
FROM raw_steps rs
INNER JOIN trials ON trials.og_id = rs.trial_id
WHERE trials.og_id IS NOT NULL; -- all trials with og_id are target trials.
"""
        cur.execute(query)  # type: ignore
        replicated_steps = cur.rowcount

        validations_cols = get_table_col_names("raw_validations") - {"id", "trial_id"}
        validations_cols_str = ", ".join(validations_cols)
        prefixed_validations_cols = ", ".join([f"rv.{col}" for col in validations_cols])
        query = f"""
-- replicate raw_validations and keep the new validation ids
INSERT INTO raw_validations( {validations_cols_str}, trial_id )
SELECT {prefixed_validations_cols}, trials.id
FROM raw_validations rv
INNER JOIN trials ON trials.og_id = rv.trial_id
WHERE trials.og_id IS NOT NULL;
"""

        cur.execute(query)  # type: ignore
        replicated_validations = cur.rowcount

        query = f""" 
-- drop the added column
ALTER TABLE {table} DROP COLUMN og_id;
"""
        cur.execute(query)  # type: ignore
        cur.execute("COMMIT")
        return {
            "trials": replicated_trials,
            "steps": replicated_steps,
            "validations": replicated_validations,
        }


if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--suffix",
        type=str,
        help="sql suffix to select the trials to replicate. eg WHERE state = 'COMPLETED' LIMIT 2",
    )
    parser.add_argument("--trial-id", type=int, default=None, help="trial id to replicate")
    args = parser.parse_args()

    assert args.suffix is None or args.trial_id is None, "cannot specify both suffix and trial_id"

    start = time.time()

    if args.trial_id is not None:
        copy_trial(args.trial_id)
        exit(0)

    row_counts = copy_trials(
        suffix=args.suffix or f"WHERE state = 'COMPLETED' LIMIT 2",
    )

    end = time.time()
    print("rows affected:", row_counts)
    print("overall time (ms):", (end - start) * 1000)
