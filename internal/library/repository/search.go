package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/sawthorn/lantern/internal/library"
)

const searchLimit = 25

type SearchResults struct {
	Tracks []library.TrackSummary `json:"tracks"`
	Albums []library.AlbumSummary `json:"albums"`
}

func (r *LibraryRepository) Search(ctx context.Context, query string) (SearchResults, error) {
	results := SearchResults{
		Tracks: make([]library.TrackSummary, 0),
		Albums: make([]library.AlbumSummary, 0),
	}

	query = strings.TrimSpace(query)
	if query == "" {
		return results, nil
	}

	// preserve wildcard symbols in query by escaping them
	escaped := strings.NewReplacer(
		"!", "!!",
		"%", "!%",
		"_", "!_",
	).Replace(query)
	pattern := "%" + escaped + "%"

	err := r.RunAsTx(ctx, func(tx *sql.Tx) error {
		trackRows, err := tx.QueryContext(ctx, `
			SELECT
				id, title, artist,
				album, album_id, track_num
			FROM tracks
			WHERE title LIKE ?1 ESCAPE '!'
			   OR artist LIKE ?1 ESCAPE '!'
			   OR album LIKE ?1 ESCAPE '!'
			   OR album_artist LIKE ?1 ESCAPE '!'
			ORDER BY title COLLATE NOCASE, id
			LIMIT ?2
		`, pattern, searchLimit)
		if err != nil {
			return fmt.Errorf("query tracks: %w", err)
		}
		defer trackRows.Close()

		for trackRows.Next() {
			if err := ctx.Err(); err != nil {
				return err
			}

			var track library.TrackSummary
			if err := trackRows.Scan(
				&track.ID, &track.Title,
				&track.Artist, &track.Album,
				&track.AlbumID, &track.TrackNum,
			); err != nil {
				return fmt.Errorf("scan track: %w", err)
			}
			results.Tracks = append(results.Tracks, track)
		}
		if err := trackRows.Err(); err != nil {
			return fmt.Errorf("iterate tracks: %w", err)
		}

		albumRows, err := tx.QueryContext(ctx, `
			SELECT
				id, title, album_artist
			FROM albums
			WHERE title LIKE ?1 ESCAPE '!'
			   OR album_artist LIKE ?1 ESCAPE '!'
			ORDER BY title COLLATE NOCASE, id
			LIMIT ?2
		`, pattern, searchLimit)
		if err != nil {
			return fmt.Errorf("query albums: %w", err)
		}
		defer albumRows.Close()

		for albumRows.Next() {
			if err := ctx.Err(); err != nil {
				return err
			}

			var album library.AlbumSummary
			if err := albumRows.Scan(
				&album.ID, &album.Title, &album.AlbumArtist,
			); err != nil {
				return fmt.Errorf("scan album: %w", err)
			}
			results.Albums = append(results.Albums, album)
		}
		if err := albumRows.Err(); err != nil {
			return fmt.Errorf("iterate albums: %w", err)
		}

		return nil
	})
	if err != nil {
		return SearchResults{}, fmt.Errorf("search library: %w", err)
	}

	return results, nil
}
