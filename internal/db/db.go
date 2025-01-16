package db

import (
	"database/sql"
	"kindExport/internal/config"
	"kindExport/internal/scrape"
	"os"
	"sync"

	. "kindExport/generated/table"

	_ "modernc.org/sqlite"
)

var (
	instance *sql.DB
	once     sync.Once
	initErr  error
)

// GetDB returns a singleton instance of the database connection
func GetDB() (*sql.DB, error) {
	if instance == nil {
		once.Do(func() {
			instance, initErr = initDB()
		})
	}
	return instance, initErr
}

func InsertBook(book scrape.Book) {
	db, err := GetDB()
	if err != nil {
		return
	}

	_ = db

	_, err = Articles.
		INSERT(Articles.Title, Articles.LocalPath, Articles.URL, Articles.Paid, Articles.Author, Articles.ReleaseDate).
		VALUES(book.Book.Title(), book.Path, book.Permalink, book.Paid, book.Book.Author(), book.ReleaseDate).
		Exec(db)

	if err != nil {
		return
	}
}

// initDB creates the initial database connection
func initDB() (*sql.DB, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", cfg.DatabasePath)

	if err != nil {
		return nil, err
	}

	_, err = db.Exec("PRAGMA journal_mode=WAL;")
	// Check whether we find the initial tables file
	for _, file := range []string{"./tables_initial.sql", "./sql/tables_initial.sql"} {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		_, err = db.Exec(string(content))
		if err != nil {
			return nil, err
		}
		break
	}

	if err != nil {
		return nil, err
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

// Close closes the database connection
// Should be called when shutting down your application
func Close() error {
	if instance != nil {
		return instance.Close()
	}
	return nil
}
