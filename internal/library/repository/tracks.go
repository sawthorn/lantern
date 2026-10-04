package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/sawthorn/lantern/internal/library"
)

func (r *LibraryRepository) GetTrackByID(ctx context.Context, id uint16) (library.Track, error) {
	res := library.Track{}
	var mtime int64
	err := r.db.QueryRowContext(ctx, `
		SELECT
			id, title,
			artist, album,
			mtime, size, path
		FROM tracks
		WHERE id=?`,
		id).Scan(
		&res.ID,
		&res.Metadata.Title,
		&res.Metadata.Artist,
		&res.Metadata.Album,
		&mtime,
		&res.FSInfo.Size,
		&res.FSInfo.Path,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return library.Track{}, library.ErrTrackNotFound
	}
	if err != nil {
		return library.Track{}, fmt.Errorf("get track: %w", err)
	}

	res.FSInfo.ModTime = time.Unix(mtime, 0)
	_, res.FSInfo.Filename = filepath.Split(res.FSInfo.Path)

	return res, nil
}

func (r *LibraryRepository) GetAllTracks(ctx context.Context) ([]library.TrackSummary, error) {
	tracks := []library.TrackSummary{}
	err := r.RunAsTx(ctx, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, `
		SELECT id, title, artist, album, album_id, track_num
		FROM tracks
	`)
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

			tracks = append(tracks, tr)
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("row iteration err: %w", err)
		}
		return nil
	})
	if err != nil {
		return []library.TrackSummary{}, fmt.Errorf("get tracks: %w", err)
	}
	return tracks, nil
}
