"""One-shot DB initializer.

Reads schema.sql (idempotent — uses CREATE TABLE IF NOT EXISTS) and
applies it against $DATABASE_URL. Safe to re-run.

Invoke from Railway:
    railway ssh --service ocx-api -- python3 /opt/ocx/init_db.py
"""
from __future__ import annotations

import os
import sys
from pathlib import Path

import psycopg


def main() -> int:
    url = os.environ.get("DATABASE_URL")
    if not url:
        print("ERROR: DATABASE_URL not set", file=sys.stderr)
        return 1

    schema_path = Path(__file__).parent / "schema.sql"
    if not schema_path.exists():
        # Container layout: schema.sql sits next to init_db.py at /opt/ocx
        alt = Path("/opt/ocx/schema.sql")
        if alt.exists():
            schema_path = alt
        else:
            print(f"ERROR: schema.sql not found at {schema_path}", file=sys.stderr)
            return 1

    sql = schema_path.read_text()
    print(f"applying {schema_path} ({len(sql)} bytes)")

    with psycopg.connect(url, autocommit=True) as conn:
        with conn.cursor() as cur:
            cur.execute(sql)
            cur.execute(
                "SELECT tablename FROM pg_tables "
                "WHERE schemaname = current_schema() ORDER BY tablename"
            )
            tables = [r[0] for r in cur.fetchall()]

    print(f"OK. tables in schema: {tables}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
