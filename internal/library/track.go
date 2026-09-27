package library

import (
	"time"

	"go.senan.xyz/taglib"
)

type AlbumCover struct {
	taglib.ImageDesc
	Data []byte // Raw picture data.
}

type Track struct {
	ID       uint16        `json:"id"`
	Metadata TrackMetadata `json:"metadata"`
	FSInfo   TrackFSInfo   `json:"fs_info"`
}

type TrackMetadata struct {
	Title       string      `json:"title"`
	Artist      string      `json:"artist"`
	Album       string      `json:"album"`
	AlbumArtist string      `json:"album_artist"`
	TrackNum    uint16      `json:"track"`
	Year        uint16      `json:"-"`
	Cover       *AlbumCover `json:"-"`
	AlbumID     uint16      `json:"album_id,omitempty"`
}

type TrackFSInfo struct {
	Path     string    `json:"path"`
	Filename string    `json:"filename"`
	ModTime  time.Time `json:"modtime"`
	Size     int64     `json:"size"`
}

// TrackSummary represents a compact view of track's crucial metadata.
type TrackSummary struct {
	ID       uint16 `json:"id"`
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Album    string `json:"album"`
	AlbumID  uint16 `json:"album_id,omitempty"`
	TrackNum uint16 `json:"track"`
}

func (a TrackSummary) RankID() uint16 {
	return a.ID
}
