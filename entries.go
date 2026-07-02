package main

import (
	"context"
	"fmt"
	"os"
	"time"
	"github.com/jackc/pgx/v5"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
	"github.com/charmbracelet/lipgloss/table"
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
	fmt.Println(entryTextStyle.Render("Entry saved as ") + idStyle.Render(fmt.Sprintf("#%d", id)) + entryTextStyle.Render("!"))
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
		return Entry{}, fmt.Errorf("No entry found with id #%d.", id)
	}
	if err != nil {
		return Entry{}, err
	}
	return entry, nil
}

func getBoxWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width <= 0 {
		return 40 // fallback if size can't be detected (e.g. piped output)
	}
	if width > 100 {
		width = 100 // optional cap so boxes don't get absurdly wide on huge terminals
	}
	return width - 4 // leave room for border + padding
}

func listEntries(conn *pgx.Conn) {
	entries, err := loadEntries(conn)
	if err != nil {
		fmt.Println("Error loading entries:", err)
		return
	}

	if len(entries) == 0 {
		fmt.Println("No entries yet.")
		return
	}

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#cdf1fa")).
		Bold(true).
		Padding(0, 1)

	evenRowStyle := lipgloss.NewStyle().Padding(1, 1)

	oddRowStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("250")).
		Padding(1, 1)

	var rows [][]string
	for _, e := range entries {
		preview := e.Contents
		if len(preview) > 60 {
			preview = preview[:60] + "..."
		}
		rows = append(rows, []string{
			fmt.Sprintf("#%d", e.ID),
			e.Date.Format("2 Jan 2006 15:04"),
			preview,
		})
	}

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("#446971"))).
		StyleFunc(func(row, col int) lipgloss.Style {
			switch {
			case row == table.HeaderRow:
				return headerStyle
			case row%2 == 0:
				return evenRowStyle
			default:
				return oddRowStyle
			}
		}).
		Headers("ID", "Date", "Preview").
		Rows(rows...)

	fmt.Println(t)
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
	// clears entries and also restarts serial count for entry IDs
	_, err := conn.Exec(context.Background(),
		`TRUNCATE TABLE entries RESTART IDENTITY`,
	)
	return err
}