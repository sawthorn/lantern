package web

import (
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"os"
	"strconv"

	clog "github.com/charmbracelet/log"
	"github.com/sawthorn/lantern/internal/library"
)

func SetUpRouting(s *MusicServer) *http.ServeMux {
	router := http.NewServeMux()

	fs := http.FileServer(http.FS(s.staticFiles))
	router.Handle(
		"GET /static/",
		http.StripPrefix("/static/", fs),
	)
	router.HandleFunc("GET /{$}", s.handleHome)
	router.HandleFunc("GET /tracks", s.handleTrackList)
	router.HandleFunc("GET /albums", s.handleAlbumList)
	router.HandleFunc("GET /api/stream/{id}", s.handleStream)
	router.HandleFunc("GET /api/albums/{id}", s.handleAlbumByID)
	router.HandleFunc("GET /api/tracks", s.handleTracks)
	router.HandleFunc("GET /api/cover/{id}", s.handleCover)
	router.HandleFunc("GET /api/albums", s.handleAlbums)
	router.HandleFunc("GET /api/tracks/{id}/download", s.handleTrackDownload)
	// router.HandleFunc("GET /api/search", s.handleSearch)

	return router
}

func (s *MusicServer) handleHome(w http.ResponseWriter, r *http.Request) {
	files := []string{
		"base.html",
		"nav.html",
		"player.html",
		"home.html",
	}
	tmpl, err := template.ParseFS(s.staticFiles, files...)
	if err != nil {
		clog.Errorf("couldn't parse templates: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = tmpl.Execute(w, nil)
	if err != nil {
		clog.Errorf("couldn't execute templates: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *MusicServer) handleStream(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		log.Errorf("handleStream: %s", err)
		http.Error(w, "ID must be an integer", http.StatusBadRequest)
		return
	}

	track, err := s.repo.GetTrackByID(uint16(id))
	if err != nil {
		log.Infof("404 Response for track #%d. %s", id, err)
		http.NotFound(w, r)
		return
	}

	f, err := os.Open(track.FSInfo.Path)
	if err != nil {
		log.Errorf("handleStream: %s", err)
		http.Error(w, "cannot open file", http.StatusInternalServerError)
	}
	defer f.Close()
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeContent(w, r, track.FSInfo.Filename, track.FSInfo.ModTime, f)
}

func (s *MusicServer) handleTracks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	tracks, err := s.repo.GetAllTracks()
	if err != nil {
		clog.Errorf("get album list json: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(tracks)
	if err != nil {
		log.Errorf("handleTracks: %s", err)
		http.Error(w, "cannot serialize Tracks objects", http.StatusInternalServerError)
	}
}

func (s *MusicServer) handleTrackList(w http.ResponseWriter, r *http.Request) {
	files := []string{
		"base.html",
		"nav.html",
		"player.html",
		"tracks_page.html",
	}
	tmpl, err := template.ParseFS(s.staticFiles, files...)
	if err != nil {
		clog.Errorf("couldn't parse templates: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = tmpl.Execute(w, nil)
	if err != nil {
		clog.Errorf("couldn't execute templates: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *MusicServer) handleCover(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		log.Errorf("handleCover: %s", err)
		http.Error(w, "ID must be an integer", http.StatusBadRequest)
		return
	}

	cover, err := s.coverCache.GetAlbumCover(uint16(id))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", cover.MIMEType)
	w.Write(cover.Data)
}

func (s *MusicServer) handleAlbums(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	albums, err := s.repo.GetAlbums()
	if err != nil {
		clog.Errorf("get album list json: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(albums)
	if err != nil {
		log.Errorf("handleAlbums: %s", err)
		http.Error(w, "cannot serialize Album objects", http.StatusInternalServerError)
	}
}

func (s *MusicServer) handleAlbumByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		log.Errorf("handleStream: %s", err)
		http.Error(w, "ID must be an integer", http.StatusBadRequest)
		return
	}

	albums, err := s.repo.GetAlbumByID(uint16(id))
	if errors.Is(err, library.ErrAlbumNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		clog.Errorf("get album list json: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(albums)
	if err != nil {
		log.Errorf("handleAlbums: %s", err)
		http.Error(w, "cannot serialize Album objects", http.StatusInternalServerError)
	}
}

func (s *MusicServer) handleAlbumList(w http.ResponseWriter, r *http.Request) {
	files := []string{
		"base.html",
		"nav.html",
		"player.html",
		"albums_page.html",
	}
	tmpl, err := template.ParseFS(s.staticFiles, files...)
	if err != nil {
		clog.Errorf("couldn't parse templates: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = tmpl.Execute(w, nil)
	if err != nil {
		clog.Errorf("couldn't execute templates: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *MusicServer) handleTrackDownload(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		log.Errorf("handleStream: %s", err)
		http.Error(w, "ID must be an integer", http.StatusBadRequest)
		return
	}

	track, err := s.repo.GetTrackByID(uint16(id))
	if err != nil {
		log.Errorf("get track: %s", err)
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Disposition", "attachment; filename="+track.FSInfo.Path)
	w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
	http.ServeFile(w, r, track.FSInfo.Path)
}

// func (s *MusicServer) handleSearch(w http.ResponseWriter, r *http.Request) {
// 	query := r.URL.Query().Get("q")
// 	tracks := s.search(query)

// 	err := json.NewEncoder(w).Encode(tracks)
// 	if err != nil {
// 		log.Errorf("handleSearch: %s", err)
// 		http.Error(w, "cannot serialize Tracks objects", http.StatusInternalServerError)
// 	}
// }
