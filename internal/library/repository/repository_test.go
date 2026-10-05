package repository

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/sawthorn/lantern/internal/library"
	"github.com/stretchr/testify/require"
)

func TestWithDBDir_Correctness(t *testing.T) {
	dbPath := t.TempDir()

	repo, err := NewLibraryRepository(
		WithDBDir(dbPath),
	)

	require.NoError(t, err)
	require.Equal(t, repo.dbPath, filepath.Join(dbPath, "library.db"))
	require.Nil(t, repo.coverCache)
}

func TestWithCoverCache_Correctness(t *testing.T) {
	cc, err := library.NewCoverCache(
		library.WithCachePath(t.TempDir()),
	)
	require.NoError(t, err)

	dbPath := t.TempDir()
	repo, err := NewLibraryRepository(
		WithDBDir(dbPath), // .db will be removed after test
		WithCoverCache(cc),
	)

	require.NoError(t, err)
	require.Equal(t, repo.dbPath, filepath.Join(dbPath, "library.db"))
	require.Equal(t, repo.coverCache, cc)
}

func TestCreateSchema_Success(t *testing.T) {
	repo := newTestRepository(t)

	t.Run("can insert album", func(t *testing.T) {
		_, err := repo.db.Exec(`
			INSERT INTO albums(title, album_artist)
			VALUES("Album", "AlbumArtist")
		`)
		require.NoError(t, err)
	})

	t.Run("albums unique constraint", func(t *testing.T) {
		_, err := repo.db.Exec(`
			INSERT INTO albums(title, album_artist)
			VALUES("Album", "AlbumArtist")
		`)
		require.Error(t, err)
	})

	t.Run("can insert track", func(t *testing.T) {
		_, err := repo.db.Exec(`
			INSERT INTO tracks(title, album_artist, path)
			VALUES("Track", "AlbumArtist", "dir/file.mp3")
		`)
		require.NoError(t, err)
	})

	t.Run("tracks unique constraint", func(t *testing.T) {
		_, err := repo.db.Exec(`
			INSERT INTO tracks(title, album_artist, path)
			VALUES("Track", "AlbumArtist", "dir/file.mp3")
		`)
		require.Error(t, err)
	})
}

func TestRunAsTx_RollbackOnError(t *testing.T) {
	repo := newTestRepository(t)

	expectedErr := errors.New("alarm")

	err := repo.RunAsTx(context.Background(), func(tx *sql.Tx) error {
		_, err := tx.Exec(`
			INSERT INTO albums(title, album_artist)
			VALUES("Album", "AlbumArtist")
		`)
		require.NoError(t, err)

		return expectedErr
	})

	require.Error(t, err)

	var count int
	err = repo.db.QueryRow(`
		SELECT COUNT(*) FROM albums
		`).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, count, 0)
}

func TestRunAsTx_CommitsOnSuccess(t *testing.T) {
	repo := newTestRepository(t)

	err := repo.RunAsTx(context.Background(), func(tx *sql.Tx) error {
		_, err := tx.Exec(`
			INSERT INTO albums(title, album_artist)
			VALUES("Album", "AlbumArtist")
		`)
		require.NoError(t, err)

		return nil
	})

	require.NoError(t, err)

	var count int
	err = repo.db.QueryRow(`
		SELECT COUNT(*) FROM albums
		`).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, count, 1)
}

func TestRunAsTx_ContextCancelled(t *testing.T) {
	repo := newTestRepository(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := repo.RunAsTx(ctx, func(tx *sql.Tx) error {
		t.Fatal("callback shouldn't run")
		return nil
	})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

// newTestRepository creates and initializes
// new `LibraryRepository` object
// with `.db` placed in test's temporary dir
func newTestRepository(t *testing.T) *LibraryRepository {
	t.Helper()

	repo, err := NewLibraryRepository(
		WithDBDir(t.TempDir()),
	)
	require.NoError(t, err)

	err = repo.Init(context.Background())
	require.NoError(t, err)

	return repo
}
