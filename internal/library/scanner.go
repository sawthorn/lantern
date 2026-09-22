package library

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/charmbracelet/log"
	"go.senan.xyz/taglib"
)

type metadata map[string][]string

var audioTypes = []string{".mp3", ".wav", ".ogg", ".flac"}

func isSupportedAudio(name string) bool {
	ext := filepath.Ext(name)
	if ext == "" {
		return false
	}

	return slices.Contains(audioTypes, ext)
}

func FullScan(dir string, repoInsertFn func(Track) error) error {
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !isSupportedAudio(d.Name()) {
			return nil
		}

		track, err := parseTrack(path, d)
		if err != nil {
			log.Printf("skipping %s: %v", path, err)
			return nil
		}

		err = repoInsertFn(track)
		if err != nil {
			return fmt.Errorf("insert: %w", err)
		}

		return nil
	})

	return err
}

func SyncScan(dir string, cachedTracks []TrackFSInfo, upsertFn func(Track) error, deleteFn func(path string) error) error {
	cachedTracksMap := make(map[string]TrackFSInfo, len(cachedTracks))

	for _, t := range cachedTracks {
		cachedTracksMap[t.Path] = t
	}

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !isSupportedAudio(d.Name()) {
			return nil
		}

		cachedTrack, exists := cachedTracksMap[path]

		delete(cachedTracksMap, path)

		info, err := d.Info()
		// dont care about err - file renamed or moved
		// if renamed - will be upserted to repo as "new"
		// if moved(deleted) - will be deleted from cache
		if err != nil {
			return nil
		}

		if !exists ||
			cachedTrack.ModTime.Unix() != info.ModTime().Unix() ||
			cachedTrack.Size != info.Size() {

			parsedTrack, err := parseTrack(path, d)
			if err != nil {
				log.Printf("skipping %s: %v", path, err)
				return nil
			}
			err = upsertFn(parsedTrack)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	for _, t := range cachedTracksMap {
		if err = deleteFn(t.Path); err != nil {
			break
		}
	}

	return err
}

func firstInTag(m metadata, key string) string {
	v, ok := m[key]
	if !ok || len(v) == 0 {
		return ""
	}
	return strings.TrimSpace(v[0])
}

func extractTrackNum(m metadata) uint16 {
	if v := firstInTag(m, taglib.TrackNumber); v != "" {
		trackStr, _, _ := strings.Cut(v, "/")
		trNum, err := strconv.ParseUint(trackStr, 10, 16)
		if err != nil {
			return 0
		}
		return uint16(trNum)
	}
	return 0
}

func extractYear(m metadata) uint16 {
	v := firstInTag(m, taglib.Date)

	if len(v) >= 4 {
		v = v[:4]
	}

	year, err := strconv.ParseUint(v, 10, 16)
	if err != nil {
		return 0
	}

	return uint16(year)
}

func extractAlbumArtist(m metadata) string {
	if v := firstInTag(m, taglib.AlbumArtist); v != "" {
		return v
	}

	return firstInTag(m, taglib.Artist)
}

func parseTrack(path string, entry os.DirEntry) (Track, error) {
	tr, err := entry.Info()
	if err != nil {
		return Track{}, fmt.Errorf("get file info: %w", err)
	}

	m, err := taglib.ReadTags(path)
	if err != nil {
		return Track{}, fmt.Errorf("tag reading: %w", err)
	}

	res := Track{
		Metadata: TrackMetadata{
			Title:       firstInTag(m, taglib.Title),
			Artist:      firstInTag(m, taglib.Artist),
			Album:       firstInTag(m, taglib.Album),
			AlbumArtist: extractAlbumArtist(m),
			TrackNum:    extractTrackNum(m),
			Year:        extractYear(m),
		},
		FSInfo: TrackFSInfo{
			Path:     path,
			Filename: tr.Name(),
			ModTime:  tr.ModTime(),
			Size:     tr.Size(),
		},
	}
	return res, nil
}
