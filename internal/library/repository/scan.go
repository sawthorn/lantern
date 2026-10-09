package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	clog "github.com/charmbracelet/log"
	"github.com/sawthorn/lantern/internal/library"
)

const upsertTrackSQL = `INSERT INTO tracks (
	title, artist, album, album_artist, track_num, year, album_id, mtime, size, path
	)
	VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT (path)
	DO UPDATE SET
		title = excluded.title,
		artist = excluded.artist,
		album = excluded.album,
		album_artist = excluded.album_artist,
		track_num = excluded.track_num,
		year = excluded.year,
		album_id = excluded.album_id,
		mtime = excluded.mtime,
		size = excluded.size
	`

func (r *LibraryRepository) ImportLibrary(ctx context.Context, rootPath string) error {
	return library.FullScan(rootPath, r.saveScannedTrack)
}

func (r *LibraryRepository) Sync(ctx context.Context, rootPath string) error {
	cachedTracks, err := r.getTracksFSInfo(ctx)
	if err != nil {
		return err
	}

	// TODO: context
	return library.SyncScan(
		rootPath,
		cachedTracks,
		r.saveScannedTrack,
		r.deleteTrackByPath,
	)
}

func (r *LibraryRepository) getTracksFSInfo(ctx context.Context) ([]library.TrackFSInfo, error) {
	// Select a len of tracks and allocate slice cap
	tracks := []library.TrackFSInfo{}
	rows, err := r.db.QueryContext(ctx, `
		SELECT path, mtime, size
		FROM tracks
	`)
	if err != nil {
		return []library.TrackFSInfo{}, fmt.Errorf("bulk select tracks: %w", err)
	}
	defer rows.Close()

	var mtime int64
	for rows.Next() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		tr := library.TrackFSInfo{}
		err = rows.Scan(&tr.Path, &mtime, &tr.Size)
		if err != nil {
			return []library.TrackFSInfo{}, fmt.Errorf("row scan err: %w", err)
		}

		tr.ModTime = time.Unix(mtime, 0)
		_, tr.Filename = filepath.Split(tr.Path)

		tracks = append(tracks, tr)
	}
	if err := rows.Err(); err != nil {
		return []library.TrackFSInfo{}, fmt.Errorf("row iteration err: %w", err)
	}

	return tracks, nil
}

func (r *LibraryRepository) saveScannedTrack(track library.Track) error {
	var albumID uint16
	var created bool
	ctx := context.Background() // TODO: reconsider
	err := r.RunAsTx(ctx, func(tx *sql.Tx) error {
		var err error
		albumID, created, err = getOrCreateAlbum(ctx, tx, track)
		if err != nil {
			return err
		}

		track.Metadata.AlbumID = &albumID

		_, err = tx.ExecContext(ctx, upsertTrackSQL,
			track.Metadata.Title, track.Metadata.Artist,
			track.Metadata.Album, track.Metadata.AlbumArtist,
			track.Metadata.TrackNum, track.Metadata.Year,
			track.Metadata.AlbumID,
			track.FSInfo.ModTime.Unix(),
			track.FSInfo.Size, track.FSInfo.Path,
		)
		return err
	})
	if err != nil {
		return err
	}

	if created && r.coverCache != nil {
		if err := r.coverCache.CacheAlbumCover(
			track.FSInfo.Path, albumID,
		); err != nil {
			clog.Warnf("cache album cover %s: %v", *track.Metadata.Album, err)
		}
	}

	return nil
}

func (r *LibraryRepository) deleteTrackByPath(path string) error {
	_, err := r.db.Exec(`DELETE FROM tracks WHERE path = ?`, path)
	return err
}

func getOrCreateAlbum(ctx context.Context, tx *sql.Tx, track library.Track) (uint16, bool, error) {
	var id uint16
	err := tx.QueryRowContext(ctx, `
		SELECT id
		FROM albums
		WHERE title = ?
		AND album_artist = ?
	`, track.Metadata.Album, track.Metadata.AlbumArtist).Scan(&id)

	// nil or something other than "Not Found"
	if !errors.Is(err, sql.ErrNoRows) {
		return id, false, err
	}

	err = tx.QueryRowContext(ctx, `
		INSERT INTO albums(title, album_artist)
		VALUES (?, ?)
		RETURNING id
	`, track.Metadata.Album, track.Metadata.AlbumArtist).Scan(&id)
	return id, true, err
}
