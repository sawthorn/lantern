package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/sawthorn/lantern/internal/library"
)

func (r *LibraryRepository) GetAlbumByID(ctx context.Context, id uint16) (library.AlbumDetails, error) {
	album := library.AlbumDetails{}
	err := r.RunAsTx(ctx, func(tx *sql.Tx) error {
		err := tx.QueryRowContext(ctx, `
		SELECT
			a.id, a.title,
			a.album_artist,
			MIN(year) AS year,
			COUNT(*) AS total_tracks
		FROM albums a
		LEFT JOIN tracks t ON a.id = t.album_id
		WHERE a.id = ?
	`, id).Scan(&album.ID, &album.Title, &album.AlbumArtist, &album.Year, &album.TotalTracks)
		if errors.Is(err, sql.ErrNoRows) {
			return library.ErrAlbumNotFound
		}
		if err != nil {
			return fmt.Errorf("album query: %w", err)
		}

		fields := "id, title, artist, album, album_id, track_num"
		q := fmt.Sprintf("SELECT %s FROM tracks WHERE album_id = ? ORDER BY track_num", fields)
		rows, err := tx.QueryContext(ctx, q, id)
		if err != nil {
			return fmt.Errorf("bulk select tracks: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			if err := ctx.Err(); err != nil {
				return err
			}

			tr := library.TrackSummary{}
			err = rows.Scan(
				&tr.ID, &tr.Title,
				&tr.Artist, &tr.Album,
				&tr.AlbumID, &tr.TrackNum,
			)
			if err != nil {
				return fmt.Errorf("row scan err: %w", err)
			}

			album.Tracks = append(album.Tracks, tr)
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("row iteration err: %w", err)
		}

		return nil
	})
	if err != nil {
		return library.AlbumDetails{}, fmt.Errorf("get album: %w", err)
	}
	return album, nil
}

func (r *LibraryRepository) GetAlbums(ctx context.Context) ([]library.Album, error) {
	albums := []library.Album{}
	rows, err := r.db.QueryContext(ctx, `
	SELECT
		a.id, a.title,
		a.album_artist,
		MIN(year) AS year,
		COUNT(*) AS total_tracks
	FROM albums a
	LEFT JOIN tracks t ON a.id = t.album_id
	GROUP BY a.title, a.album_artist
	ORDER BY a.id
	`)
	if err != nil {
		return nil, fmt.Errorf("query execution: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		al := library.Album{}
		err = rows.Scan(
			&al.ID, &al.Title,
			&al.AlbumArtist,
			&al.Year, &al.TotalTracks,
		)
		if err != nil {
			return nil, fmt.Errorf("row scan err: %w", err)
		}

		albums = append(albums, al)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration err: %s", err)
	}

	return albums, nil
}
