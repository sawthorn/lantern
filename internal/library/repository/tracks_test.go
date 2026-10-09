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

func TestGetTrackByID_TrackHasSomeDataSuccess(t *testing.T) {
	repo := newTestRepository(t)

	seedAlbums(t, 10, repo)
	_, err := repo.db.Exec(`
		INSERT INTO tracks(id, title, path, mtime, size)
		VALUES(1, "Test", "path/test/track.mp3", 1, 1)
		`)
	require.NoError(t, err)
	seedTracksForAlbum(t, 5, 1, repo)
	seedTracksForAlbum(t, 5, 2, repo)

	track, err := repo.GetTrackByID(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, uint16(1), track.ID)
	assert.Equal(t, "Test", *track.Metadata.Title)
	assert.Equal(t, "path/test/track.mp3", track.FSInfo.Path)
}

func TestGetTrackByID_Missing(t *testing.T) {
	repo := newTestRepository(t)

	track, err := repo.GetTrackByID(context.Background(), 1)
	require.Error(t, err)
	require.Equal(t, library.Track{}, track)
}

func TestGetTrackByID_CancelledContext(t *testing.T) {
	repo := newTestRepository(t)

	_, err := repo.db.Exec(`
		INSERT INTO tracks(id, title, path, mtime, size)
		VALUES(1, "Test", "path/test/track.mp3", 1, 1)
		`)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = repo.GetTrackByID(ctx, 1)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want: context.Canceled, got: %v", err)
	}
}

func TestGetTrackByID_TrackHasAllData(t *testing.T) {
	repo := newTestRepository(t)

	seedAlbums(t, 10, repo)
	_, err := repo.db.Exec(`
		INSERT INTO tracks(
			id, title, artist,
			album, album_artist,
			track_num, year, album_id,
			mtime, size, path)
		VALUES(
			1, "Test", "Artist",
			"Album", "Album's Artist",
			13, 2017, 1,
			1, 1, "path/test/13.mp3")
		`)
	require.NoError(t, err)
	seedTracksForAlbum(t, 5, 1, repo)
	seedTracksForAlbum(t, 5, 2, repo)

	track, err := repo.GetTrackByID(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, uint16(1), track.ID)
	assert.Equal(t, "Test", *track.Metadata.Title)
	assert.Equal(t, "Artist", *track.Metadata.Artist)
	assert.Equal(t, int64(1), track.FSInfo.ModTime.Unix())
	assert.Equal(t, int64(1), track.FSInfo.Size)
	assert.Equal(t, "path/test/13.mp3", track.FSInfo.Path)
}

func TestGetAllTracks_Success(t *testing.T) {
	repo := newTestRepository(t)

	seedAlbums(t, 10, repo)

	_, err := repo.db.Exec(`
		INSERT INTO tracks(
		id,  title, artist,
		album, album_id, track_num, path)
		SELECT 1 AS id, "Test1" AS title,
			"Artist1" AS artist, "Album1" AS album,
			1 AS album_id, 1 AS track_num, "t1" AS path
		UNION SELECT 2 AS id, "Test2" AS title,
			"Artist2" AS artist, "Album2" AS album,
			2 AS album_id, 2 AS track_num, "t2" AS path
		UNION SELECT 3 AS id, "Test3" AS title,
			"Artist3" AS artist, "Album3" AS album,
			3 AS album_id, 3 AS track_num, "t3" AS path
		`)
	require.NoError(t, err)

	tracks, err := repo.GetAllTracks(context.Background())
	require.NoError(t, err)
	require.Len(t, tracks, 3)

	for i, tr := range tracks {
		assert.Equal(t, uint16(i+1), tr.ID)
		assert.Equal(t, fmt.Sprintf("Test%d", i+1), *tr.Title)
		assert.Equal(t, fmt.Sprintf("Artist%d", i+1), *tr.Artist)
		assert.Equal(t, fmt.Sprintf("Album%d", i+1), *tr.Album)
		assert.Equal(t, uint16(i+1), *tr.AlbumID)
		assert.Equal(t, uint16(i+1), *tr.TrackNum)
	}
}

func TestGetAllTracks_NoTracksInDB(t *testing.T) {
	repo := newTestRepository(t)

	tracks, err := repo.GetAllTracks(context.Background())
	require.NoError(t, err)
	require.Len(t, tracks, 0)
}

func TestGetAllTracks_CancelledContext(t *testing.T) {
	repo := newTestRepository(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	res, err := repo.GetAllTracks(ctx)
	require.Len(t, res, 0)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want: context.Canceled, got: %v", err)
	}
}
