package web

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/sawthorn/lantern/internal/library"
	"github.com/stretchr/testify/require"
)

func TestRank_SizeAndUniqueness(t *testing.T) {
	today := time.Date(2026, time.March, 13, 8, 0, 0, 0, time.UTC)

	selectionSize := 7

	testcases := []struct {
		name       string
		albumCount int
	}{
		{name: "empty library", albumCount: 0},
		{name: "one album", albumCount: 1},
		{name: "fewer than limit", albumCount: selectionSize / 2},
		{name: "exactly limit", albumCount: selectionSize},
		{name: "more than limit", albumCount: selectionSize * 2},
	}

	for _, tc := range testcases {
		albums := mockAlbums(tc.albumCount)

		picks := rank(albums, selectionSize, today)

		wantCount := min(tc.albumCount, selectionSize)
		require.Equal(t, len(picks), wantCount, fmt.Sprintf("selected %d, want %d", len(picks), wantCount))

		if tc.albumCount == 0 {
			require.Nil(t, picks, "picks not nil for empty library")
		} else {
			require.NotNil(t, picks, "picks nil for non-empty library")
		}

		seen := make(map[uint16]struct{}, len(picks))

		for _, elem := range picks {
			_, exists := seen[elem.ID]
			require.False(t, exists, fmt.Sprintf("album %d picked twice", elem.ID))

			seen[elem.ID] = struct{}{}

			// sanity check for data mutation
			require.False(t, elem.ID == 0 || int(elem.ID) > tc.albumCount, fmt.Sprintf("selected album %d not present in input", elem.ID))
		}
	}
}

func TestRank_ConsistentDuringSameDay(t *testing.T) {
	albums := mockAlbums(30)

	morning := time.Date(2026, time.March, 13, 8, 0, 0, 0, time.UTC)
	evening := time.Date(2026, time.March, 13, 18, 0, 0, 0, time.UTC)

	firstRanked := rank(albums, homeAlbumSelectionSize, morning)
	secondRanked := rank(albums, homeAlbumSelectionSize, evening)

	require.Equal(t, firstRanked, secondRanked)
}

func TestRank_IndependentOfInputOrder(t *testing.T) {
	albums := mockAlbums(30)

	today := time.Date(2026, time.March, 13, 8, 0, 0, 0, time.UTC)
	reversed := append([]library.Album{}, albums...)
	slices.Reverse(reversed)

	originalRanked := rank(albums, homeAlbumSelectionSize, today)
	reversedRanked := rank(reversed, homeAlbumSelectionSize, today)

	require.Equal(t, originalRanked, reversedRanked)
}

func TestRank_DoesNotModifyInput(t *testing.T) {
	today := time.Date(2026, time.September, 10, 0, 0, 0, 0, time.UTC)

	albums := mockAlbums(20)
	original := append([]library.Album(nil), albums...)

	rank(albums, homeAlbumSelectionSize, today)

	require.Equal(t, albums, original)
}

func mockAlbums(count int) []library.Album {
	albums := make([]library.Album, count)

	for i := range count {
		albums[i] = library.Album{
			ID:    uint16(i + 1),
			Title: fmt.Sprintf("Album %d", i+1),
		}
	}

	return albums
}
