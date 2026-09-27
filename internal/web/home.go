package web

import (
	"net/http"
	"time"

	clog "github.com/charmbracelet/log"
	"github.com/sawthorn/lantern/internal/library"
)

const (
	homeAlbumSelectionSize = 7
	homeTrackSelectionSize = 10
)

type homePageData struct {
	AlbumCount int
	TrackCount int
	Featured   *library.Album
	Albums     []library.Album
	Tracks     []library.TrackSummary
}

func (s *MusicServer) homePage(w http.ResponseWriter, r *http.Request) {
	nowTimestamp := time.Now()

	hpd := homePageData{}
	albums, err := s.repo.GetAlbums()
	if err != nil {
		clog.Errorf("get albums from repo: %s", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	hpd.AlbumCount = len(albums)
	albumPicks := rank(albums, homeAlbumSelectionSize, nowTimestamp)
	hpd.Featured, hpd.Albums = &albumPicks[0], albumPicks[1:]

	tracks, err := s.repo.GetAllTracks()
	if err != nil {
		clog.Errorf("get tracks from repo: %s", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	hpd.TrackCount = len(tracks)
	hpd.Tracks = rank(tracks, homeTrackSelectionSize, nowTimestamp)

	err = s.views.Render(w, "home", hpd)
	if err != nil {
		clog.Errorf("render home: %s", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}
