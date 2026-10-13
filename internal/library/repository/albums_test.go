package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/sawthorn/lantern/internal/library"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAlbumByID_NoTracksExist(t *testing.T) {
	repo := newTestRepository(t)
	seedAlbums(t, 10, repo)

	album, err := repo.GetAlbumByID(context.Background(), uint16(2))
	require.NoError(t, err)
	assert.Equal(t, uint16(2), album.ID)
	assert.Equal(t, "Album 2", *album.Title)
	assert.Equal(t, "Artist 2", *album.AlbumArtist)
}

func TestGetAlbumByID_TracksExist(t *testing.T) {
	repo := newTestRepository(t)
	seedAlbums(t, 10, repo)
	seedTracksForAlbum(t, 10, 1, repo)
	seedTracksForAlbum(t, 10, 2, repo)

	album, err := repo.GetAlbumByID(context.Background(), uint16(2))
	require.NoError(t, err)
	assert.Equal(t, uint16(2), album.ID)
	assert.Equal(t, "Album 2", *album.Title)
	assert.Equal(t, "Artist 2", *album.AlbumArtist)
	assert.Len(t, album.Tracks, 10)
	for _, tr := range album.Tracks {
		assert.Equal(t, uint16(2), *tr.AlbumID)
	}
}

func TestGetAlbumByID_CancelledContext(t *testing.T) {
	repo := newTestRepository(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	album, err := repo.GetAlbumByID(ctx, 0)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
	require.Equal(t, library.AlbumDetails{}, album)
}

func TestGetAlbums_Success(t *testing.T) {
	repo := newTestRepository(t)
	seedAlbums(t, 10, repo)

	albums, err := repo.GetAlbums(context.Background())
	require.NoError(t, err)
	require.Len(t, albums, 10)
}

func TestGetAlbums_EmptyDB(t *testing.T) {
	repo := newTestRepository(t)

	albums, err := repo.GetAlbums(context.Background())
	require.NoError(t, err)
	require.Empty(t, albums)
}

func TestGetAlbums_NullArtist(t *testing.T) {
	repo := newTestRepository(t)
	_, err := repo.db.Exec(`
		INSERT INTO albums(title)
		VALUES("Foo")
		`)
	require.NoError(t, err)

	_, err = repo.db.Exec(`
		INSERT INTO albums(title, album_artist)
		SELECT "DAMN" AS title,  "Kendrick Lamar" AS album_artist
		UNION SELECT "Promises" AS title, "Floating Points" AS album_artist
		`)
	require.NoError(t, err)

	albums, err := repo.GetAlbums(context.Background())

	require.NoError(t, err)
	require.Len(t, albums, 3)

	require.Equal(t, "Foo", *albums[0].Title)
	require.Nil(t, albums[0].AlbumArtist)
}

func TestGetAlbums_CancelledContext(t *testing.T) {
	repo := newTestRepository(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	albums, err := repo.GetAlbums(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
	require.Nil(t, albums)
}

func seedAlbums(t *testing.T, count int, repo *LibraryRepository) {
	t.Helper()

	var max_ind int
	row := repo.db.QueryRow(`SELECT COALESCE(MAX(id), 0) FROM albums`)
	err := row.Scan(&max_ind)
	require.NoError(t, err)

	for i := max_ind + 1; i <= max_ind+count; i++ {
		_, err := repo.db.Exec(`
		INSERT INTO albums(id, title, album_artist)
		VALUES(?1, ?2, ?3)
		`,
			i,
			fmt.Sprintf("Album %d", i),
			fmt.Sprintf("Artist %d", i),
		)
		require.NoError(t, err)
	}
}

func seedTracksForAlbum(t *testing.T, count int, album_id uint16, repo *LibraryRepository) {
	t.Helper()

	for i := range count {
		_, err := repo.db.Exec(`
		INSERT INTO tracks(title, path, album_id)
		VALUES(?1, ?2, ?3)
		`,
			fmt.Sprintf("Track%d-%d", album_id, i),
			fmt.Sprintf("path/%d/%d", album_id, i),
			album_id,
		)
		require.NoError(t, err)
	}
}
