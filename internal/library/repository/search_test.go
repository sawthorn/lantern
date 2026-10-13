package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearch_Success(t *testing.T) {
	repo := newTestRepository(t)
	repo.db.Exec(`
		INSERT INTO albums(id, title, album_artist)
		VALUES(1, "SearchTA", "TestArtist")
		`)
	repo.db.Exec(`
		INSERT INTO tracks(id, title, album, album_id, path)
		SELECT 1 AS id, "T1" AS title, "SearchTA" AS album, 1 AS album_id, "/ta/t1" AS path
		UNION SELECT 2 AS id, "T2" AS title, "SearchTA" AS album, 1 AS album_id, "/ta/t2" AS path 
		`)
	seedAlbums(t, 10, repo)
	seedTracksForAlbum(t, 2, 2, repo)
	seedTracksForAlbum(t, 2, 5, repo)

	sr, err := repo.Search(context.Background(), "SearchTA")
	require.NoError(t, err)
	require.Len(t, sr.Tracks, 2)
	require.Len(t, sr.Albums, 1)
	album := sr.Albums[0]
	assert.Equal(t, uint16(1), album.ID)
	assert.Equal(t, "SearchTA", *album.Title)
	assert.Equal(t, "TestArtist", *album.AlbumArtist)
	for i, tr := range sr.Tracks {
		assert.Equal(t, uint16(i+1), tr.ID)
		assert.Equal(t, uint16(1), *tr.AlbumID)
		assert.Equal(t, fmt.Sprintf("T%d", i+1), *tr.Title)
		assert.Equal(t, "SearchTA", *tr.Album)
	}
}

func TestSearch_QueryFormattingWorks(t *testing.T) {
	repo := newTestRepository(t)

	repo.db.Exec(`
		INSERT INTO albums(title)
		SELECT "SearchTA" AS title
		UNION SELECT "ESCAPED!" AS title
		UNION SELECT "CAse" as title
		`)

	testcases := []struct {
		name     string
		query    string
		expected string
	}{
		{name: "excess spaces", query: "   SearchTA    ", expected: "SearchTA"},
		{name: "escaping", query: "ESCAPED!", expected: "ESCAPED!"},
		{name: "case-sensitive", query: "caSE", expected: "CAse"},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			sr, err := repo.Search(context.Background(), tc.query)
			assert.NoError(t, err)
			assert.Len(t, sr.Albums, 1)
			assert.Len(t, sr.Tracks, 0)
			assert.Equal(t, tc.expected, *sr.Albums[0].Title)
		})
	}
}

func TestSearch_NoResults(t *testing.T) {
	repo := newTestRepository(t)
	seedAlbums(t, 10, repo)
	sr, err := repo.Search(context.Background(), "")
	require.NoError(t, err)
	assert.Len(t, sr.Albums, 0)
	assert.Len(t, sr.Tracks, 0)
}

func TestSearch_NoResultsRepoEmpty(t *testing.T) {
	repo := newTestRepository(t)

	sr, err := repo.Search(context.Background(), "Test")
	require.NoError(t, err)
	assert.Len(t, sr.Albums, 0)
	assert.Len(t, sr.Tracks, 0)
}

func TestSearch_CanceledContext(t *testing.T) {
	repo := newTestRepository(t)

	seedAlbums(t, 10, repo)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	searchRes, err := repo.Search(ctx, "1")
	require.Len(t, searchRes.Albums, 0)
	require.Len(t, searchRes.Tracks, 0)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want: context.Canceled, got: %v", err)
	}
}
