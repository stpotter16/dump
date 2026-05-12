package db

import (
	"context"
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"runtime"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	writeDB *sql.DB
	readDB  *sql.DB
}

func New(directory string) (DB, error) {
	if err := ensureDirectorExists(directory); err != nil {
		return DB{}, err
	}

	dbPath := filepath.Join(directory, "coin.sqlite")
	readDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Printf("Could not open read db: %v", err)
		return DB{}, err
	}
	readDB.SetMaxOpenConns(max(4, runtime.NumCPU()))
	if err = applyPragmas(readDB); err != nil {
		log.Printf("Could not apply PRAGMAs to read db: %v", err)
		return DB{}, err
	}

	writeDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Printf("Could not open write db: %v", err)
		return DB{}, err
	}
	writeDB.SetMaxOpenConns(1)
	if err = applyPragmas(writeDB); err != nil {
		log.Printf("Could not apply PRAGMAs to write db: %v", err)
		return DB{}, err
	}

	db := DB{
		readDB:  readDB,
		writeDB: writeDB,
	}

	return db, nil
}

func (db *DB) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	rows, err := db.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		log.Printf("Could not execute query: %v", err)
		return nil, err
	}
	return rows, nil
}

func (db *DB) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return db.readDB.QueryRowContext(ctx, query, args...)
}

func (db *DB) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	result, err := db.writeDB.ExecContext(ctx, query, args...)
	if err != nil {
		log.Printf("Could not execute statement: %v", err)
		return nil, err
	}
	return result, nil
}

func (db *DB) WithTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := db.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (db DB) ExecuteTransaction(ctx context.Context, transactions ...string) error {
	tx, err := db.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	for _, statement := range transactions {
		_, err = tx.Exec(statement)
		if err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

func applyPragmas(db *sql.DB) error {
	if _, err := db.Exec(`
		-- https://litestream.io/tips/
		-- https://kerkour.com/sqlite-for-servers
		PRAGMA journal_mode = WAL;
		PRAGMA busy_timeout = 5000;
		PRAGMA synchronous = NORMAL;
        PRAGMA wal_autocheckpoint = 0;
		PRAGMA cache_size = 1000000000;
		PRAGMA foreign_keys = true;
		PRAGMA temp_store = memory;
	`); err != nil {
		return err
	}
	return nil
}

func ensureDirectorExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			return err
		}
	}
	return nil
}
