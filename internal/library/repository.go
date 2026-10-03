package library

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

	searchLimit = 25
)

type LibraryRepository struct {
	db         *sql.DB
	dbPath     string
	coverCache *CoverCache
}

type LibraryRepositoryOption func(*LibraryRepository)

var errEmptyDBPath = errors.New("empty DB path")

func NewLibraryRepository(opts ...LibraryRepositoryOption) (*LibraryRepository, error) {
	defaultPath, err := defaultDBPath()
	if err != nil {
		return nil, fmt.Errorf("get default DB path: %w", err)
	}
	r := &LibraryRepository{dbPath: defaultPath, coverCache: nil}

	for _, opt := range opts {
		opt(r)
	}

	if r.dbPath == "" {
		return nil, errEmptyDBPath
	}

	if err := os.MkdirAll(filepath.Dir(r.dbPath), 0750); err != nil {
		return nil, fmt.Errorf("create db parent dir: %w", err)
	}

	r.db, err = sql.Open("sqlite", r.dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db connection: %w", err)
	}

	return r, nil
}

func WithDBPath(path string) LibraryRepositoryOption {
	return func(r *LibraryRepository) {
		r.dbPath = filepath.Join(path, "library.db")
	}
}

func WithCoverCache(cc *CoverCache) LibraryRepositoryOption {
	return func(r *LibraryRepository) {
		r.coverCache = cc
	}
}

func defaultDBPath() (string, error) {
	cacheDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(cacheDir, ".local/share/lantern/library.db"), nil
}

func (r *LibraryRepository) NeedsImport() bool {
	_, err := os.Stat(r.dbPath)
	if errors.Is(err, os.ErrNotExist) {
		return true
	}
	return err != nil
}

func (r *LibraryRepository) createSchema(ctx context.Context) error {
	return r.RunAsTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
	CREATE TABLE IF NOT EXISTS albums(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT,
		album_artist TEXT
	)
	`)
		if err != nil {
			return fmt.Errorf("create albums table: %w", err)
		}

		_, err = tx.ExecContext(ctx, `
	CREATE UNIQUE INDEX idx_album_unique
	ON albums(title, album_artist);
	`)
		if err != nil {
			return fmt.Errorf("create unique albums index: %w", err)
		}

		_, err = tx.ExecContext(ctx, `
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
	})
}

func (r *LibraryRepository) GetTrackByID(ctx context.Context, id uint16) (Track, error) {
	res := Track{}
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
		return Track{}, ErrTrackNotFound
	}
	if err != nil {
		return Track{}, fmt.Errorf("get track: %w", err)
	}

	res.FSInfo.ModTime = time.Unix(mtime, 0)
	_, res.FSInfo.Filename = filepath.Split(res.FSInfo.Path)

	return res, nil
}

func (r *LibraryRepository) GetAlbums(ctx context.Context) ([]Album, error) {
	albums := []Album{}
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

func (r *LibraryRepository) GetAlbumByID(ctx context.Context, id uint16) (AlbumDetails, error) {
	album := AlbumDetails{}
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
			return ErrAlbumNotFound
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

			tr := TrackSummary{}
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
		return AlbumDetails{}, fmt.Errorf("get album: %w", err)
	}
	return album, nil
}

func (r *LibraryRepository) GetAllTracks(ctx context.Context) ([]TrackSummary, error) {
	tracks := []TrackSummary{}
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

			tr := TrackSummary{}
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
		return []TrackSummary{}, fmt.Errorf("get tracks: %w", err)
	}
	return tracks, nil
}

func (r *LibraryRepository) ImportLibrary(ctx context.Context, rootPath string) error {
	err := r.createSchema(ctx)
	if err != nil {
		return fmt.Errorf("create schema: %w", err)
	}

	return FullScan(rootPath, r.saveScannedTrack)
}

func (r *LibraryRepository) Sync(ctx context.Context, rootPath string) error {
	cachedTracks, err := r.getTracksFSInfo(ctx)
	if err != nil {
		return err
	}

	// TODO: context
	return SyncScan(
		rootPath,
		cachedTracks,
		r.saveScannedTrack,
		r.deleteTrackByPath,
	)
}

func (r *LibraryRepository) RunAsTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
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

type SearchResults struct {
	Tracks []TrackSummary `json:"tracks"`
	Albums []AlbumSummary `json:"albums"`
}

func (r *LibraryRepository) Search(ctx context.Context, query string) (SearchResults, error) {
	results := SearchResults{
		Tracks: make([]TrackSummary, 0),
		Albums: make([]AlbumSummary, 0),
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

			var track TrackSummary
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

			var album AlbumSummary
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

func (r *LibraryRepository) getTracksFSInfo(ctx context.Context) ([]TrackFSInfo, error) {
	// Select a len of tracks and allocate slice cap
	tracks := []TrackFSInfo{}
	rows, err := r.db.QueryContext(ctx, `
		SELECT path, mtime, size
		FROM tracks
	`)
	if err != nil {
		return []TrackFSInfo{}, fmt.Errorf("bulk select tracks: %w", err)
	}
	defer rows.Close()

	var mtime int64
	for rows.Next() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

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
	ctx := context.Background() // TODO: reconsider
	err := r.RunAsTx(ctx, func(tx *sql.Tx) error {
		var err error
		albumID, created, err = getOrCreateAlbum(ctx, tx, track)
		if err != nil {
			return err
		}

		track.Metadata.AlbumID = albumID

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
			clog.Warnf("cache album cover %s: %v", track.Metadata.Album, err)
		}
	}

	return nil
}

func (r *LibraryRepository) deleteTrackByPath(path string) error {
	_, err := r.db.Exec(`DELETE FROM tracks WHERE path = ?`, path)
	return err
}

func getOrCreateAlbum(ctx context.Context, tx *sql.Tx, track Track) (uint16, bool, error) {
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
