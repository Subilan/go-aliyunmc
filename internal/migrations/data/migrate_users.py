#!/usr/bin/env python3
"""
Copy rows from old `users` table to the new-schema table.
Source and target can be in different databases on the same server.

Usage:
  python3 migrate_users.py --host 127.0.0.1 --user root --password xxx \
      --source-db old_db --target-db new_db

  python3 migrate_users.py --source-db db1 --source-table users_old \
      --target-db db2 --target-table users

Env vars: MYSQL_HOST, MYSQL_USER, MYSQL_PASSWORD
"""

import argparse
import os
import sys

try:
    import pymysql
except ImportError:
    print("Missing pymysql. Install it: pip install pymysql")
    sys.exit(1)


def main():
    p = argparse.ArgumentParser(description="Migrate rows from old users table to new")
    p.add_argument("--host", default=os.environ.get("MYSQL_HOST", "127.0.0.1"))
    p.add_argument("--port", type=int, default=int(os.environ.get("MYSQL_PORT", "3306")))
    p.add_argument("--user", default=os.environ.get("MYSQL_USER", "root"))
    p.add_argument("--password", default=os.environ.get("MYSQL_PASSWORD", ""))
    p.add_argument("--source-db", required=True, help="source database name")
    p.add_argument("--source-table", default="users", help="source table name")
    p.add_argument("--target-db", required=True, help="target database name")
    p.add_argument("--target-table", default="users", help="target table name")
    p.add_argument("--dry-run", action="store_true", help="print SQL without executing")
    args = p.parse_args()

    src = f"`{args.source_db}`.`{args.source_table}`"
    tgt = f"`{args.target_db}`.`{args.target_table}`"

    insert_sql = f"""
INSERT INTO {tgt} (`id`, `created_at`, `updated_at`, `deleted_at`, `username`, `password_hash`, `role`, `whitelist_uuid`)
SELECT `id`, `created_at`, NULL, NULL, `username`, `password_hash`, '', NULL
FROM {src}
""".strip()

    if args.dry_run:
        print(insert_sql)
        return

    conn = pymysql.connect(
        host=args.host,
        port=args.port,
        user=args.user,
        password=args.password,
        charset="utf8mb4",
    )
    cur = conn.cursor()

    try:
        cur.execute(f"SELECT COUNT(*) FROM {src}")
        src_count = cur.fetchone()[0]
        print(f"Source {src}: {src_count} rows")

        cur.execute(f"SELECT COUNT(*) FROM {tgt}")
        tgt_before = cur.fetchone()[0]
        print(f"Target {tgt} (before): {tgt_before} rows")

        cur.execute(insert_sql)
        conn.commit()
        print(f"Inserted: {cur.rowcount} rows")

        cur.execute(f"SELECT COUNT(*) FROM {tgt}")
        tgt_after = cur.fetchone()[0]
        if tgt_after != tgt_before + src_count:
            print(f"ERROR: count mismatch. Expected {tgt_before + src_count}, got {tgt_after}")
            sys.exit(1)

        print(f"Target {tgt} (after): {tgt_after} rows")
        print("Done.")

    except Exception as e:
        conn.rollback()
        print(f"Error: {e}")
        sys.exit(1)
    finally:
        cur.close()
        conn.close()


if __name__ == "__main__":
    main()
