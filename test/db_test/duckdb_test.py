#! /usr/bin/env python3

import duckdb

print("creating a DuckDB DB file - 'test_duckdb.db'")
conn = duckdb.connect("test_duckdb.db")

print("creating a table")
conn.sql("CREATE TABLE movie(title VARCHAR, year INTEGER, score DOUBLE)")

print("inserting records to the table")
data = [
    ("Monty Python Live at the Hollywood Bowl", 1982, 7.9),
    ("Monty Python The Meaning of Life", 1983, 7.5),
    ("Monty Python Life of Brian", 1979, 8.0),
]
for d in data:
    conn.execute("INSERT INTO movie VALUES('%s', %d, %f)" % d)

print("retrieving records from the table")
conn.execute("SELECT * FROM movie")
for row in conn.fetchall():
    print(row)

print("done!")
