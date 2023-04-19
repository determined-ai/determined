#!/usr/bin/env python

"""
connect to Determined's db and replicate various metrics for performance testing purposes.
"""

import concurrent.futures
import contextlib
import os
import time
from typing import Any, Dict, Generator, List, Set, Tuple, Union

import psycopg  # pip install "psycopg[binary]"
from psycopg import sql

DB_NAME = os.environ.get("DET_DB_NAME", "determined")
DB_USERNAME = os.environ.get("DET_DB_USERNAME", "postgres")
DB_PASSWORD = os.environ.get("DET_DB_PASSWORD", "postgres")
DB_HOST = os.environ.get("DET_DB_HOST", "localhost")
DB_PORT = os.environ.get("DET_DB_PORT", "5432")

Query = Union[str, bytes, sql.SQL, sql.Composable]


# a class extending psycopg.Cursor that adds logging around each query execute.
class LoggingCursor(psycopg.Cursor):  # type: ignore
    def execute(self, query: Query, *args: Any, **kwargs: Any) -> "LoggingCursor":
        print(
            f"""====QUERY START====
{query.strip() if isinstance(query, str) else str(query)}
====QUERY END===="""
        )
        start = time.time()
        super().execute(query, *args, **kwargs)
        end = time.time()
        print("query took (ms):", (end - start) * 1000)
        return self


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
    cur = conn.cursor()
    yield cur
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


def replicate_rows(table: str, skip_cols: Set[str], multiplier: int = 1, suffix: str = "") -> None:
    cols = get_table_col_names(table) - skip_cols
    cols_str = ", ".join(cols)

    with db_cursor() as cur:
        query = f"""
INSERT INTO {table}( {cols_str} )
SELECT {cols_str} FROM {table}
CROSS JOIN generate_series(1, {multiplier}) AS g
{suffix};
        """
        cur.execute(query)
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
        cur.execute(query, (trial_id, trial_id, trial_id))
        cur.execute("COMMIT")


def submit_db_queries(
    cursor: psycopg.Cursor, queries: List[Tuple[str, str]]
) -> Generator[Tuple[str, int], None, None]:
    """
    submit a set of db queries concurrently yield the changes as they are ready.
    queries: list of (name, query)
    """
    with concurrent.futures.ThreadPoolExecutor(max_workers=4) as executor:
        # submit each query and record the rows affected for each one
        def job(name: str, query: str) -> Tuple[str, int]:
            cursor.execute(query)
            return name, cursor.rowcount

        futures = [executor.submit(job, name, query) for name, query in queries]
        # process as each future ready
        for future in concurrent.futures.as_completed(futures):
            yield future.result()


@contextlib.contextmanager
def duplicate_table_rows(
    cur: psycopg.Cursor, table: str, suffix: str = ""
) -> Generator[int, None, None]:
    """
    duplicate rows of a table and keep a mapping between old and new ids.
    `id` col is assumed to auto increment.
    return: number of new rows added
    """
    cols = get_table_col_names(table) - {"id"}
    cols_str = ", ".join(cols)
    values_str = ", ".join([table + "." + col for col in cols])

    query = f"""
-- modify the target table to add a new col called og_id
ALTER TABLE {table} ADD COLUMN og_id int;
"""
    cur.execute(query)

    query = f"""
-- insert the replicated rows populating the og_id column with the original id
INSERT INTO {table}( {cols_str}, og_id )
SELECT {values_str}, {table}.id
FROM {table}
{suffix};
"""
    cur.execute(query)
    affected_rows = cur.rowcount

    query = f"""
CREATE TEMP TABLE {table}_id_map AS
SELECT id, og_id
FROM {table}
WHERE og_id IS NOT NULL;
"""
    cur.execute(query)

    yield affected_rows

    # tear down
    query = f"""
-- drop the table
-- DROP TABLE {table}_id_map; -- temp table
-- drop the added column
ALTER TABLE {table} DROP COLUMN og_id;
    """
    cur.execute(query)


@contextlib.contextmanager
def _copy_trials(
    cur: psycopg.Cursor, suffix: str = "", exclude_single_searcher: bool = True
) -> Generator[dict, None, None]:
    affected_rows: Dict[str, int] = {}
    table = "trials"
    trial_suffix = f"""
JOIN experiments e ON e.id = {table}.experiment_id
-- CROSS JOIN generate_series(1, multiplier) AS g
WHERE e.config->'searcher'->>'name' <> 'single'
{suffix};
    """
    if not exclude_single_searcher:
        trial_suffix = f"""
WHERE 1 = 1
{suffix};
        """
    with duplicate_table_rows(cur, table, suffix=trial_suffix) as added_trials:
        affected_rows["trials"] = added_trials
        if added_trials == 0:
            yield affected_rows
            return

        steps_cols = get_table_col_names("raw_steps") - {"id"} - {"trial_id"}
        steps_cols_str = ", ".join(steps_cols)
        prefixed_steps_cols = ", ".join([f"rs.{col}" for col in steps_cols])
        # replicate raw_steps and update trial_id
        steps_query = f"""

-- replicate raw_steps and keep the new step ids
INSERT INTO raw_steps( {steps_cols_str}, trial_id )
SELECT {prefixed_steps_cols}, {table}_id_map.id
FROM raw_steps rs
INNER JOIN {table}_id_map ON {table}_id_map.og_id = rs.trial_id
-- WHERE {table}_id_map.og_id IS NOT NULL; -- all {table}_id_map with og_id are target trials.
"""
        cur.execute(steps_query)
        affected_rows["steps"] = cur.rowcount

        validations_cols = get_table_col_names("raw_validations") - {"id", "trial_id"}
        validations_cols_str = ", ".join(validations_cols)
        prefixed_validations_cols = ", ".join([f"rv.{col}" for col in validations_cols])
        validations_query = f"""
-- replicate raw_validations and keep the new validation ids
INSERT INTO raw_validations( {validations_cols_str}, trial_id )
SELECT {prefixed_validations_cols}, {table}_id_map.id
FROM raw_validations rv
INNER JOIN {table}_id_map ON {table}_id_map.og_id = rv.trial_id
-- WHERE {table}_id_map.og_id IS NOT NULL;
"""

        cur.execute(validations_query)
        affected_rows["validations"] = cur.rowcount
        yield affected_rows

    cur.execute("COMMIT")


def copy_trials(suffix: str = "", exclude_single_searcher: bool = True) -> dict:
    """
    Duplicate trials and associated metrics for multi trial experiments.
    """
    with db_cursor() as cur:
        with _copy_trials(cur, suffix, exclude_single_searcher) as affected_rows:
            return affected_rows


def copy_experiments() -> dict:
    """
    - copy experiments, keep id mapping
    - copy trials and all metrics and keep id mapping
    - update trials' experiment_id to the new experiment id
    """
    added_rows: Dict[str, int] = {}
    with db_cursor() as cur:
        with duplicate_table_rows(cur, "experiments") as added_exps:
            added_rows["experiments"] = added_exps
            if added_exps == 0:
                return added_rows
            with _copy_trials(cur, exclude_single_searcher=False) as affected_rows:
                added_rows.update(affected_rows)
                """
                tables
                - experiments: id
                - experiments_id_map: id, og_id
                - trials: id
                - trials_id_map: id, og_id
                """
                # update new trials' experiment_id to the newly added experiment id
                query = """
UPDATE trials
SET experiment_id = sub.new_exp_id
FROM (
    SELECT trials.id as trial_id, experiments_id_map.id AS new_exp_id
    FROM trials
    JOIN trials_id_map ON trials.id = trials_id_map.id
    JOIN experiments_id_map ON trials.experiment_id = experiments_id_map.og_id
) as sub
WHERE trials.id = sub.trial_id;
"""
                cur.execute(query)
        cur.execute("COMMIT")
    return added_rows


if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser()
    parser.description = "Replicate trials within experiments for multi-trial experiments\
        or whole experiments at db level in bulk."

    parser.add_argument("mode", type=str, help="mode to run in: trials, experiments")
    parser.add_argument(
        "--suffix",
        type=str,
        default="",
        help="sql suffix to select the trials to replicate this appends after an existing\
        WHERE clause. eg AND state = 'COMPLETED' LIMIT 2",
    )
    parser.add_argument("--trial-id", type=int, default=None, help="trial id to replicate")
    parser.add_argument(
        "--naive-multiplier", type=int, default=None, help="repeat the operation n times (naive)"
    )
    args = parser.parse_args()

    assert args.suffix == "" or args.trial_id is None, "cannot specify both suffix and trial_id"
    assert args.mode in ["trials", "experiments"], "mode must be either trials or experiments"

    start = time.time()

    row_counts = None
    for _ in range(args.naive_multiplier or 1):
        if args.mode == "experiments":
            assert args.trial_id is None, "cannot specify trial_id in experiments mode"
            assert args.suffix == ""
            row_counts = copy_experiments()
        elif args.mode == "trials":
            if args.trial_id is not None:
                copy_trial(args.trial_id)
            else:
                counts = copy_trials(suffix=args.suffix)
                row_counts = (
                    {k: v + counts[k] for k, v in row_counts.items()}
                    if row_counts is not None
                    else counts
                )
        else:
            raise ValueError(f"unknown mode: {args.mode}")

    end = time.time()
    print("rows added:", row_counts)
    print("overall time (ms):", (end - start) * 1000)
