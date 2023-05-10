#!/usr/bin/env python3

"""
test the reversibility of a migration by running it in a transaction both ways and
timing it.
"""

import contextlib
import os
import pathlib
import time
import subprocess
import tempfile
from typing import Generator, List, Tuple

import psycopg  # pip install "psycopg[binary]"

DB_NAME = os.environ.get("DET_DB_NAME", "determined")
DB_USERNAME = os.environ.get("DET_DB_USERNAME", "postgres")
DB_PASSWORD = os.environ.get("DET_DB_PASSWORD", "postgres")
DB_HOST = os.environ.get("DET_DB_HOST", "localhost")
DB_PORT = os.environ.get("DET_DB_PORT", "5432")

MIGRATIONS_DIR = pathlib.Path("master/static/migrations")


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
            print("Rolling back the transaction")
            cur.execute("ROLLBACK")
            raise
        else:
            cur.execute("COMMIT")


def generate_schema(dbname: str = DB_NAME) -> pathlib.Path:
    with tempfile.NamedTemporaryFile(mode="w", delete=False) as temp_file:
        command = ["pg_dump", "-U", DB_USERNAME, "-h", DB_HOST, "-p", DB_PORT, "-s", dbname]
        subprocess.run(command, stdout=temp_file)
        return pathlib.Path(temp_file.name)


def run_migration(name: str, statements: str) -> None:
    with db_transaction() as cur:
        start = time.time()
        cur.execute(statements)  # type: ignore
        end = time.time()
        print(f"Ran {name} in {end - start:.2f}s")


def get_migration_paths(query: str) -> List[Tuple[pathlib.Path, pathlib.Path]]:
    migration_files = list(MIGRATIONS_DIR.glob(f"*{query}*"))
    assert len(migration_files) % 2 == 0, f"expected even files for {query} got {migration_files}"
    up_files = [f for f in migration_files if "up" in f.name]
    down_files = [f for f in migration_files if "down" in f.name]
    return list(zip(sorted(up_files), sorted(down_files)))


def diff_files(old_path: pathlib.Path, new_path: pathlib.Path) -> str:
    out = subprocess.run(["diff", old_path, new_path], check=True)
    return out.stdout.decode("utf-8") if out.stdout else ""


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
    migration_files = get_migration_paths(name)
    assert len(migration_files) == 1, f"expected one migration for {name} got {migration_files}"
    up_file, down_file = migration_files[0]
    assert up_file.exists(), f"{up_file} does not exist"
    assert down_file.exists(), f"{down_file} does not exist"
    name = up_file.name.split(".up")[0].split("_", 1)[1]
    migration_files = [up_file, down_file] if not reverse else [down_file, up_file]
    statements = "\n".join([f.read_text() for f in migration_files])

    old_schema_path = generate_schema()
    run_migration(name + "-merged", statements)
    new_schema_path = generate_schema()
    schema_diff = diff_files(old_schema_path, new_schema_path)
    assert schema_diff == "", f"{up_file} did not reverse cleanly: \n\n{schema_diff}"


if __name__ == "__main__":
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
