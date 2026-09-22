package library

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	clog "github.com/charmbracelet/log"
	_ "modernc.org/sqlite"
)

const (
	upsertTrackSQL = `INSERT INTO tracks (
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
)

type LibraryRepository struct {
	db         *sql.DB
	dbPath     string
	coverCache *CoverCache
}

func NewLibraryRepository(dbPath string, coverCache *CoverCache) (*LibraryRepository, error) {
	var err error
	r := LibraryRepository{dbPath: dbPath, coverCache: coverCache}
	r.db, err = sql.Open("sqlite", r.dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db connection: %w", err)
	}

	return &r, nil
}

func (r *LibraryRepository) NeedsImport() bool {
	_, err := os.Stat(r.dbPath)
	if errors.Is(err, os.ErrNotExist) {
		return true
	}
	return err != nil
}

func createSchema(tx *sql.Tx) error {
	_, err := tx.Exec(`
	CREATE TABLE IF NOT EXISTS albums(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT,
		album_artist TEXT
	)
	`)
	if err != nil {
		return fmt.Errorf("create albums table: %w", err)
	}

	_, err = tx.Exec(`
	CREATE UNIQUE INDEX idx_album_unique
	ON albums(title, album_artist);
	`)
	if err != nil {
		return fmt.Errorf("create unique albums index: %w", err)
	}

	_, err = tx.Exec(`
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
}

func (r *LibraryRepository) GetTrackByID(id uint16) (Track, error) {
	res := Track{}
	var mtime int64
	err := r.db.QueryRow(`
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
		return Track{}, ErrTrackNotFound
	}
	if err != nil {
		return Track{}, fmt.Errorf("get track: %w", err)
	}

	res.FSInfo.ModTime = time.Unix(mtime, 0)
	_, res.FSInfo.Filename = filepath.Split(res.FSInfo.Path)

	return res, nil
}

func (r *LibraryRepository) GetAlbums() ([]Album, error) {
	albums := []Album{}
	rows, err := r.db.Query(`
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
		al := Album{}
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

func (r *LibraryRepository) GetAlbumByID(id uint16) (AlbumDetails, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return AlbumDetails{}, fmt.Errorf("transaction begin: %w", err)
	}
	defer tx.Rollback()

	album := AlbumDetails{}
	err = tx.QueryRow(`
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
		return AlbumDetails{}, ErrAlbumNotFound
	}
	if err != nil {
		return AlbumDetails{}, fmt.Errorf("album query: %w", err)
	}

	fields := "id, title, artist, album, album_id, track_num"
	q := fmt.Sprintf("SELECT %s FROM tracks WHERE album_id = ? ORDER BY track_num", fields)
	rows, err := tx.Query(q, id)
	if err != nil {
		return AlbumDetails{}, fmt.Errorf("bulk select tracks: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		tr := TrackSummary{}
		err = rows.Scan(
			&tr.ID, &tr.Title,
			&tr.Artist, &tr.Album,
			&tr.AlbumID, &tr.TrackNum,
		)
		if err != nil {
			return AlbumDetails{}, fmt.Errorf("row scan err: %w", err)
		}

		album.Tracks = append(album.Tracks, tr)
	}
	if err := rows.Err(); err != nil {
		return AlbumDetails{}, fmt.Errorf("row iteration err: %w", err)
	}

	return album, tx.Commit()
}

func (r *LibraryRepository) GetAllTracks() ([]TrackSummary, error) {
	tracks := []TrackSummary{}
	tx, err := r.db.Begin()
	if err != nil {
		return []TrackSummary{}, fmt.Errorf("transaction begin: %w", err)
	}
	defer tx.Rollback()

	rows, err := tx.Query(`
		SELECT id, title, artist, album, album_id, track_num
		FROM tracks
	`)
	if err != nil {
		return []TrackSummary{}, fmt.Errorf("bulk select tracks: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		tr := TrackSummary{}
		err = rows.Scan(
			&tr.ID, &tr.Title,
			&tr.Artist, &tr.Album,
			&tr.AlbumID, &tr.TrackNum,
		)
		if err != nil {
			return []TrackSummary{}, fmt.Errorf("row scan err: %w", err)
		}

		tracks = append(tracks, tr)
	}
	if err := rows.Err(); err != nil {
		return []TrackSummary{}, fmt.Errorf("row iteration err: %w", err)
	}

	return tracks, tx.Commit()
}

func (r *LibraryRepository) ImportLibrary(rootPath string) error {
	err := r.RunAsTx(createSchema)
	if err != nil {
		return fmt.Errorf("create schema: %w", err)
	}

	return FullScan(rootPath, r.saveScannedTrack)
}

func (r *LibraryRepository) Sync(rootPath string) error {
	cachedTracks, err := r.getTracksFSInfo()
	if err != nil {
		return err
	}

	return SyncScan(
		rootPath,
		cachedTracks,
		r.saveScannedTrack,
		r.deleteTrackByPath,
	)
}

func (r *LibraryRepository) RunAsTx(fn func(tx *sql.Tx) error) error {
	tx, err := r.db.Begin()
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

func (r *LibraryRepository) getTracksFSInfo() ([]TrackFSInfo, error) {
	// Select a len of tracks and allocate slice cap
	tracks := []TrackFSInfo{}
	rows, err := r.db.Query(`
		SELECT path, mtime, size
		FROM tracks
	`)
	if err != nil {
		return []TrackFSInfo{}, fmt.Errorf("bulk select tracks: %w", err)
	}
	defer rows.Close()

	var mtime int64
	for rows.Next() {
		tr := TrackFSInfo{}
		err = rows.Scan(&tr.Path, &mtime, &tr.Size)
		if err != nil {
			return []TrackFSInfo{}, fmt.Errorf("row scan err: %w", err)
		}

		tr.ModTime = time.Unix(mtime, 0)
		_, tr.Filename = filepath.Split(tr.Path)

		tracks = append(tracks, tr)
	}
	if err := rows.Err(); err != nil {
		return []TrackFSInfo{}, fmt.Errorf("row iteration err: %w", err)
	}

	return tracks, nil
}

func (r *LibraryRepository) saveScannedTrack(track Track) error {
	var albumID uint16
	var created bool
	err := r.RunAsTx(func(tx *sql.Tx) error {
		var err error
		albumID, created, err = getOrCreateAlbum(tx, track)
		if err != nil {
			return err
		}

		track.Metadata.AlbumID = albumID

		_, err = tx.Exec(upsertTrackSQL,
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

	if created {
		if err := r.coverCache.CacheAlbumCover(
			track.FSInfo.Path, albumID,
		); err != nil {
			clog.Warnf("cache album cover %s: %v", track.Metadata.Album, err)
		}
	}

	return nil
}

func (r *LibraryRepository) deleteTrackByPath(path string) error {
	_, err := r.db.Exec(`DELETE FROM tracks WHERE path = ?`, path)
	return err
}

func getOrCreateAlbum(tx *sql.Tx, track Track) (uint16, bool, error) {
	var id uint16
	err := tx.QueryRow(`
		SELECT id
		FROM albums
		WHERE title = ?
		AND album_artist = ?
	`, track.Metadata.Album, track.Metadata.AlbumArtist).Scan(&id)

	// nil or something other than "Not Found"
	if !errors.Is(err, sql.ErrNoRows) {
		return id, false, err
	}

	err = tx.QueryRow(`
		INSERT INTO albums(title, album_artist)
		VALUES (?, ?)
		RETURNING id
	`, track.Metadata.Album, track.Metadata.AlbumArtist).Scan(&id)
	return id, true, err
}
