package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sawthorn/lantern/internal/library"
	_ "modernc.org/sqlite"
)

const searchLimit = 25

type LibraryRepository struct {
	db         *sql.DB
	dbPath     string
	coverCache *library.CoverCache
}

type LibraryRepositoryOption func(*LibraryRepository)

var errEmptyDBPath = errors.New("empty DB path")

func NewLibraryRepository(opts ...LibraryRepositoryOption) (*LibraryRepository, error) {
	defaultPath, err := defaultDBPath()
	if err != nil {
		return nil, fmt.Errorf("get default DB path: %w", err)
	}
	r := &LibraryRepository{dbPath: defaultPath, coverCache: nil}

	for _, opt := range opts {
		opt(r)
	}

	if r.dbPath == "" {
		return nil, errEmptyDBPath
	}

	if err := os.MkdirAll(filepath.Dir(r.dbPath), 0750); err != nil {
		return nil, fmt.Errorf("create db parent dir: %w", err)
	}

	r.db, err = sql.Open("sqlite", r.dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db connection: %w", err)
	}

	return r, nil
}

func WithDBDir(dir string) LibraryRepositoryOption {
	return func(r *LibraryRepository) {
		r.dbPath = filepath.Join(dir, "library.db")
	}
}

func WithCoverCache(cc *library.CoverCache) LibraryRepositoryOption {
	return func(r *LibraryRepository) {
		r.coverCache = cc
	}
}

func defaultDBPath() (string, error) {
	cacheDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(cacheDir, ".local/share/lantern/library.db"), nil
}

func (r *LibraryRepository) Init(ctx context.Context) error {
	err := r.createSchema(ctx)
	if err != nil {
		return fmt.Errorf("create schema: %w", err)
	}
	return nil
}

func (r *LibraryRepository) NeedsImport() bool {
	_, err := os.Stat(r.dbPath)
	if errors.Is(err, os.ErrNotExist) {
		return true
	}
	return err != nil
}

func (r *LibraryRepository) createSchema(ctx context.Context) error {
	return r.RunAsTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
	CREATE TABLE IF NOT EXISTS albums(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT,
		album_artist TEXT
	)
	`)
		if err != nil {
			return fmt.Errorf("create albums table: %w", err)
		}

		_, err = tx.ExecContext(ctx, `
	CREATE UNIQUE INDEX IF NOT EXISTS idx_album_unique
	ON albums(title, album_artist);
	`)
		if err != nil {
			return fmt.Errorf("create unique albums index: %w", err)
		}

		_, err = tx.ExecContext(ctx, `
	CREATE TABLE IF NOT EXISTS tracks(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT,
		artist TEXT,
		album TEXT,
		album_artist TEXT,
		track_num INTEGER,
		year INTEGER,
		album_id INTEGER,
		mtime INTEGER,
		size INTEGER,
		path TEXT UNIQUE NOT NULL,
		FOREIGN KEY (album_id) REFERENCES albums(id)
	)
	`)
		if err != nil {
			return fmt.Errorf("create tracks table: %w", err)
		}

		return nil
	})
}

func (r *LibraryRepository) RunAsTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("transaction start: %w", err)
	}
	defer tx.Rollback()

	err = fn(tx)
	if err != nil {
		return err
	}

	return tx.Commit()
}
