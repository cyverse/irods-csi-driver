#!/usr/bin/env python3

"""Exercise transactional SQLite I/O against the current working directory."""

import hashlib
import sqlite3


DATABASE_PATH = "test_sqlite.db"
BATCH_SIZE = 32
BATCH_COUNT = 8
PAYLOAD_SIZE = 32 * 1024


def make_payload(document_id: int, revision: int) -> bytes:
    seed = hashlib.sha256(f"sqlite:{document_id}:{revision}".encode()).digest()
    return (seed * ((PAYLOAD_SIZE + len(seed) - 1) // len(seed)))[:PAYLOAD_SIZE]


def checksum(payload: bytes) -> str:
    return hashlib.sha256(payload).hexdigest()


def document_row(document_id: int, revision: int):
    payload = make_payload(document_id, revision)
    return document_id, f"tenant-{document_id % 4}", revision, payload, checksum(payload)


def create_schema(conn: sqlite3.Connection) -> None:
    conn.executescript(
        """
        DROP TABLE IF EXISTS audit_log;
        DROP TABLE IF EXISTS documents;
        CREATE TABLE documents (
            document_id INTEGER PRIMARY KEY,
            tenant TEXT NOT NULL,
            revision INTEGER NOT NULL,
            content BLOB NOT NULL,
            checksum TEXT NOT NULL
        );
        CREATE INDEX documents_by_tenant_revision ON documents (tenant, revision, document_id);
        CREATE TABLE audit_log (
            event_id INTEGER PRIMARY KEY AUTOINCREMENT,
            document_id INTEGER NOT NULL,
            action TEXT NOT NULL,
            revision INTEGER NOT NULL
        );
        """
    )


def write_workload(conn: sqlite3.Connection):
    expected = {}
    total_documents = BATCH_SIZE * BATCH_COUNT
    for batch in range(BATCH_COUNT):
        start = batch * BATCH_SIZE
        rows = [document_row(document_id, 1) for document_id in range(start, start + BATCH_SIZE)]
        with conn:
            conn.executemany("INSERT INTO documents VALUES (?, ?, ?, ?, ?)", rows)
            conn.executemany(
                "INSERT INTO audit_log (document_id, action, revision) VALUES (?, 'insert', ?)",
                [(row[0], row[2]) for row in rows],
            )
        expected.update({row[0]: row for row in rows})
        print(f"committed insert batch {batch + 1}/{BATCH_COUNT}")

    updated_ids = [document_id for document_id in expected if document_id % 5 == 0]
    updates = [document_row(document_id, 2) for document_id in updated_ids]
    with conn:
        conn.executemany(
            "UPDATE documents SET tenant = ?, revision = ?, content = ?, checksum = ? WHERE document_id = ?",
            [(row[1], row[2], row[3], row[4], row[0]) for row in updates],
        )
        conn.executemany(
            "INSERT INTO audit_log (document_id, action, revision) VALUES (?, 'update', ?)",
            [(row[0], row[2]) for row in updates],
        )
    expected.update({row[0]: row for row in updates})
    print(f"committed {len(updates)} updates")

    deleted_ids = [document_id for document_id in expected if document_id % 17 == 0]
    with conn:
        conn.executemany("DELETE FROM documents WHERE document_id = ?", [(document_id,) for document_id in deleted_ids])
        conn.executemany(
            "INSERT INTO audit_log (document_id, action, revision) VALUES (?, 'delete', ?)",
            [(document_id, expected[document_id][2]) for document_id in deleted_ids],
        )
    for document_id in deleted_ids:
        del expected[document_id]
    print(f"committed {len(deleted_ids)} deletes from {total_documents} inserted documents")
    return expected, total_documents + len(updates) + len(deleted_ids)


def verify_workload(conn: sqlite3.Connection, expected, expected_audit_count: int) -> None:
    row_count, byte_count = conn.execute("SELECT COUNT(*), COALESCE(SUM(length(content)), 0) FROM documents").fetchone()
    assert row_count == len(expected), (row_count, len(expected))
    assert byte_count == len(expected) * PAYLOAD_SIZE, (byte_count, len(expected) * PAYLOAD_SIZE)

    for tenant, count in conn.execute("SELECT tenant, COUNT(*) FROM documents GROUP BY tenant ORDER BY tenant"):
        print(f"indexed tenant query: {tenant} has {count} documents")
    for document_id, tenant, revision, content, digest in conn.execute(
        "SELECT document_id, tenant, revision, content, checksum FROM documents ORDER BY document_id"
    ):
        expected_row = expected[document_id]
        assert (tenant, revision, content, digest) == expected_row[1:]
        assert checksum(content) == digest

    audit_count = conn.execute("SELECT COUNT(*) FROM audit_log").fetchone()[0]
    assert audit_count == expected_audit_count, (audit_count, expected_audit_count)


def main() -> None:
    print(f"creating SQLite workload database: {DATABASE_PATH}")
    conn = sqlite3.connect(DATABASE_PATH)
    conn.execute("PRAGMA journal_mode=WAL")
    conn.execute("PRAGMA synchronous=FULL")
    create_schema(conn)
    expected, expected_audit_count = write_workload(conn)
    conn.close()

    print("reopening database and verifying persisted data")
    conn = sqlite3.connect(DATABASE_PATH)
    verify_workload(conn, expected, expected_audit_count)
    conn.close()
    print(f"verified {len(expected)} documents ({len(expected) * PAYLOAD_SIZE} bytes) successfully")


if __name__ == "__main__":
    main()
