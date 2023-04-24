#!/usr/bin/env python3

import contextlib
import os
import pathlib
import time
from typing import Dict, Generator, Set, Tuple

import psycopg  # pip install "psycopg[binary]"

DB_NAME = os.environ.get("DET_DB_NAME", "determined")
DB_USERNAME = os.environ.get("DET_DB_USERNAME", "postgres")
DB_PASSWORD = os.environ.get("DET_DB_PASSWORD", "postgres")
DB_HOST = os.environ.get("DET_DB_HOST", "localhost")
DB_PORT = os.environ.get("DET_DB_PORT", "5432")


@contextlib.contextmanager
def db_cursor() -> Generator[psycopg.Cursor, None, None]:
    conn = psycopg.connect(
        dbname=DB_NAME,
        user=DB_USERNAME,
        password=DB_PASSWORD,
        host=DB_HOST,
        port=DB_PORT,
    )
    cur = conn.cursor()
    yield cur
    conn.close()


@contextlib.contextmanager
def db_transaction():
    with db_cursor() as cur:
        try:
            cur.execute("BEGIN")
            yield cur
        except Exception:
            print("rolling back")
            cur.execute("ROLLBACK")
            raise
        else:
            cur.execute("COMMIT")


# query the current db schemas for all tables into a dict
def get_db_schemas() -> Dict[str, Set[str]]:
    with db_cursor() as cur:
        cur.execute(
            """
            SELECT table_name, column_name
            FROM information_schema.columns
            WHERE table_schema = 'public'
            """
        )
        rows = cur.fetchall()
        schemas = {}
        for table_name, column_name in rows:
            if table_name not in schemas:
                schemas[table_name] = set()
            schemas[table_name].add(column_name)
        return schemas


def run_migration(name: str, statements: str) -> None:
    with db_transaction() as cur:
        start = time.time()
        cur.execute(statements)  # type: ignore
        end = time.time()
        print(f"Ran {name} in {end - start:.2f}s")


def run_sql_file(file_path: pathlib.Path) -> None:
    """
    run a set of sql statements from a file in a transaction
    and time it.
    """
    with file_path.open("r") as f:
        sql_statements = f.read()
    run_migration(file_path.name, sql_statements)


def get_migration_paths(query: str) -> Tuple[pathlib.Path, pathlib.Path]:
    migration_files = list(pathlib.Path(".").glob(f"*{query}*"))
    assert len(migration_files) == 2, f"expected 2 files for {query} got {migration_files}"
    up_file = next(f for f in migration_files if "up" in f.name)
    down_file = next(f for f in migration_files if "down" in f.name)
    return up_file, down_file


def diff_dicts(old: Dict[str, Set[str]], new: Dict[str, Set[str]]) -> Dict[str, Set[str]]:
    """
    return a dict of table names to the set of changes
    """
    diff = {}
    for table_name, old_columns in old.items():
        if table_name not in new:
            diff[table_name] = old_columns
        else:
            new_columns = new[table_name]
            if old_columns != new_columns:
                diff[table_name] = old_columns ^ new_columns
    return diff


def test_det_migration(name: str, reverse: bool = False):
    """
    given a migration name:
    - construct the file path to up and down directions
    - get the current schema
    - run the up migration and time it
    - run the down migration and time it
    - get the new schema
    - compare the new schema to the old schema
    """
    up_file, down_file = get_migration_paths(name)
    assert up_file.exists(), f"{up_file} does not exist"
    assert down_file.exists(), f"{down_file} does not exist"

    migration_files = [up_file, down_file] if not reverse else [down_file, up_file]
    statements = "\n".join([f.read_text() for f in migration_files])

    old_schema = get_db_schemas()
    run_migration(name + "-merged", statements)
    # for file in migration_files:
    #     print(f"Running migration {file}")
    #     run_sql_file(file)
    new_schema = get_db_schemas()
    assert (
        old_schema == new_schema
    ), f"{up_file} is not reversible, {diff_dicts(old_schema, new_schema)}"


if __name__ == "__main__":
    """
    add cli options with argparse to with help
    - get a migration name
    - reverse flag to run migrations in reverse
    """
    import argparse

    parser = argparse.ArgumentParser(description="test timing and reversibility of migrations")
    parser.add_argument("name", type=str, help="name of migration to test")
    parser.add_argument(
        "--reverse",
        action="store_true",
        help="run the migration in reverse",
    )
    args = parser.parse_args()

    test_det_migration(args.name, args.reverse)
