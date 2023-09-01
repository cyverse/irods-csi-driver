#! /usr/bin/env python3

import sqlite3

print("creating a SQLite DB file - 'test_sqlite.db'")
conn = sqlite3.connect("test_sqlite.db")
cur = conn.cursor()

print("creating a table")
cur.execute("CREATE TABLE movie(title, year, score)")

print("inserting records to the table")
data = [
    ("Monty Python Live at the Hollywood Bowl", 1982, 7.9),
    ("Monty Python's The Meaning of Life", 1983, 7.5),
    ("Monty Python's Life of Brian", 1979, 8.0),
]
cur.executemany("INSERT INTO movie VALUES(?, ?, ?)", data)
conn.commit()

print("retrieving records from the table")
for row in cur.execute("SELECT year, title FROM movie ORDER BY year"):
    print(row)

print("done!")
