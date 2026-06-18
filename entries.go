package main

import (
	"context"
	"fmt"
	"os"
	"time"
	"github.com/jackc/pgx/v5"
)

type Entry struct {
	ID       int
	Date     time.Time
	Contents string
}

func connect() (*pgx.Conn, error) {
    host     := os.Getenv("PGHOST")
    user     := os.Getenv("PGUSER")
    password := os.Getenv("PGPASSWORD")
    dbname   := os.Getenv("PGDATABASE")
    port     := os.Getenv("PGPORT")

    url := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", user, password, host, port, dbname)
    return pgx.Connect(context.Background(), url)
}

func setupDB(conn *pgx.Conn) error {
	_, err := conn.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS entries (
			id       SERIAL PRIMARY KEY,
			date     TIMESTAMPTZ NOT NULL,
			contents TEXT NOT NULL
		)
	`)
	return err
}

func saveEntry(conn *pgx.Conn, contents string) error {
	var id int
	err := conn.QueryRow(context.Background(),
		`INSERT INTO entries (date, contents) VALUES ($1, $2) RETURNING id`,
		time.Now(), contents,
	).Scan(&id)
	if err != nil {
		return err
	}
	fmt.Printf("Entry saved as #%d!\n", id)
	return nil
}

func loadEntries(conn *pgx.Conn) ([]Entry, error) {
	rows, err := conn.Query(context.Background(),
		`SELECT id, date, contents FROM entries ORDER BY date DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return pgx.CollectRows(rows, pgx.RowToStructByName[Entry])
}

func loadOneEntry(conn *pgx.Conn, id int) (Entry, error) {
	rows, err := conn.Query(context.Background(),
		`SELECT id, date, contents FROM entries WHERE id = $1`,
		id,
	)
	if err != nil {
		return Entry{}, err
	}
	defer rows.Close()

	entry, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[Entry])
	if err == pgx.ErrNoRows {
		return Entry{}, fmt.Errorf("no entry found with id #%d", id)
	}
	if err != nil {
		return Entry{}, err
	}
	return entry, nil
}

func listEntries(conn *pgx.Conn) {
	entries, err := loadEntries(conn)
	if err != nil {
		fmt.Println("Error loading entries:", err)
		return
	}

	if len(entries) == 0 {
		fmt.Println("No entries yet!")
		return
	}

	for _, e := range entries {
		preview := e.Contents
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}
		fmt.Printf("#%d  %s\n%s\n\n", e.ID, e.Date.Format("2 Jan 2006 15:04"), preview)
	}
}

func deleteOneEntry(conn *pgx.Conn, id int) error {
	result, err := conn.Exec(context.Background(),
		`DELETE FROM entries WHERE id = $1`,
		id,
	)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("no entry found with id #%d", id)
	}

	fmt.Printf("Entry #%d has been deleted!\n", id)
	return nil
}

func clearEntries(conn *pgx.Conn) error {
	_, err := conn.Exec(context.Background(),
		`DELETE FROM entries`,
	)
	return err
}