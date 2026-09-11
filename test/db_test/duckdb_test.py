#!/usr/bin/env python3

"""Exercise transactional DuckDB I/O against the current working directory."""

import hashlib

import duckdb


DATABASE_PATH = "test_duckdb.db"
BATCH_SIZE = 32
BATCH_COUNT = 8
PAYLOAD_SIZE = 32 * 1024


def make_payload(document_id: int, revision: int) -> bytes:
    seed = hashlib.sha256(f"duckdb:{document_id}:{revision}".encode()).digest()
    return (seed * ((PAYLOAD_SIZE + len(seed) - 1) // len(seed)))[:PAYLOAD_SIZE]


def checksum(payload: bytes) -> str:
    return hashlib.sha256(payload).hexdigest()


def document_row(document_id: int, revision: int):
    payload = make_payload(document_id, revision)
    return document_id, f"tenant-{document_id % 4}", revision, payload, checksum(payload)


def create_schema(conn) -> None:
    conn.execute("DROP TABLE IF EXISTS audit_log")
    conn.execute("DROP TABLE IF EXISTS documents")
    conn.execute(
        "CREATE TABLE documents (document_id BIGINT PRIMARY KEY, tenant VARCHAR NOT NULL, revision INTEGER NOT NULL, content BLOB NOT NULL, checksum VARCHAR NOT NULL)"
    )
    conn.execute("CREATE INDEX documents_by_tenant_revision ON documents (tenant, revision, document_id)")
    conn.execute(
        "CREATE TABLE audit_log (event_id BIGINT PRIMARY KEY, document_id BIGINT NOT NULL, action VARCHAR NOT NULL, revision INTEGER NOT NULL)"
    )


def write_workload(conn):
    expected = {}
    next_event_id = 1
    total_documents = BATCH_SIZE * BATCH_COUNT
    for batch in range(BATCH_COUNT):
        start = batch * BATCH_SIZE
        rows = [document_row(document_id, 1) for document_id in range(start, start + BATCH_SIZE)]
        conn.execute("BEGIN TRANSACTION")
        conn.executemany("INSERT INTO documents VALUES (?, ?, ?, ?, ?)", rows)
        audit_rows = [(next_event_id + index, row[0], "insert", row[2]) for index, row in enumerate(rows)]
        next_event_id += len(audit_rows)
        conn.executemany("INSERT INTO audit_log VALUES (?, ?, ?, ?)", audit_rows)
        conn.execute("COMMIT")
        expected.update({row[0]: row for row in rows})
        print(f"committed insert batch {batch + 1}/{BATCH_COUNT}")

    updated_ids = [document_id for document_id in expected if document_id % 5 == 0]
    updates = [document_row(document_id, 2) for document_id in updated_ids]
    conn.execute("BEGIN TRANSACTION")
    conn.executemany(
        "UPDATE documents SET tenant = ?, revision = ?, content = ?, checksum = ? WHERE document_id = ?",
        [(row[1], row[2], row[3], row[4], row[0]) for row in updates],
    )
    audit_rows = [(next_event_id + index, row[0], "update", row[2]) for index, row in enumerate(updates)]
    next_event_id += len(audit_rows)
    conn.executemany("INSERT INTO audit_log VALUES (?, ?, ?, ?)", audit_rows)
    conn.execute("COMMIT")
    expected.update({row[0]: row for row in updates})
    print(f"committed {len(updates)} updates")

    deleted_ids = [document_id for document_id in expected if document_id % 17 == 0]
    conn.execute("BEGIN TRANSACTION")
    conn.executemany("DELETE FROM documents WHERE document_id = ?", [(document_id,) for document_id in deleted_ids])
    audit_rows = [(next_event_id + index, document_id, "delete", expected[document_id][2]) for index, document_id in enumerate(deleted_ids)]
    next_event_id += len(audit_rows)
    conn.executemany("INSERT INTO audit_log VALUES (?, ?, ?, ?)", audit_rows)
    conn.execute("COMMIT")
    for document_id in deleted_ids:
        del expected[document_id]
    print(f"committed {len(deleted_ids)} deletes from {total_documents} inserted documents")
    return expected, next_event_id - 1


def verify_workload(conn, expected, expected_audit_count: int) -> None:
    row_count, byte_count = conn.execute("SELECT COUNT(*), COALESCE(SUM(octet_length(content)), 0) FROM documents").fetchone()
    assert row_count == len(expected), (row_count, len(expected))
    assert byte_count == len(expected) * PAYLOAD_SIZE, (byte_count, len(expected) * PAYLOAD_SIZE)

    for tenant, count in conn.execute("SELECT tenant, COUNT(*) FROM documents GROUP BY tenant ORDER BY tenant").fetchall():
        print(f"indexed tenant query: {tenant} has {count} documents")
    for document_id, tenant, revision, content, digest in conn.execute(
        "SELECT document_id, tenant, revision, content, checksum FROM documents ORDER BY document_id"
    ).fetchall():
        expected_row = expected[document_id]
        assert (tenant, revision, content, digest) == expected_row[1:]
        assert checksum(content) == digest

    audit_count = conn.execute("SELECT COUNT(*) FROM audit_log").fetchone()[0]
    assert audit_count == expected_audit_count, (audit_count, expected_audit_count)


def main() -> None:
    print(f"creating DuckDB workload database: {DATABASE_PATH}")
    conn = duckdb.connect(DATABASE_PATH)
    create_schema(conn)
    expected, expected_audit_count = write_workload(conn)
    conn.close()

    print("reopening database and verifying persisted data")
    conn = duckdb.connect(DATABASE_PATH, read_only=True)
    verify_workload(conn, expected, expected_audit_count)
    conn.close()
    print(f"verified {len(expected)} documents ({len(expected) * PAYLOAD_SIZE} bytes) successfully")


if __name__ == "__main__":
    main()
