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

	staticFileServer := http.FileServer(http.FS(s.staticFiles))
	router.Handle(
		"GET /static/",
		http.StripPrefix("/static/", staticFileServer),
	)
	router.HandleFunc("GET /{$}", s.homePage)
	router.HandleFunc("GET /tracks", s.tracksPage)
	router.HandleFunc("GET /albums", s.albumsPage)
	router.HandleFunc("GET /api/stream/{id}", s.streamTrackAPI)
	router.HandleFunc("GET /api/albums/{id}", s.albumAPI)
	router.HandleFunc("GET /api/tracks", s.tracksAPI)
	router.HandleFunc("GET /api/cover/{id}", s.coverImage)
	router.HandleFunc("GET /api/albums", s.albumsAPI)
	router.HandleFunc("GET /api/tracks/{id}/download", s.downloadTrackAPI)
	// router.HandleFunc("GET /api/search", s.handleSearch)

	return router
}

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
func (s *MusicServer) homePage(w http.ResponseWriter, r *http.Request) {
	if err != nil {
		clog.Errorf("couldn't execute templates: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *MusicServer) streamTrackAPI(w http.ResponseWriter, r *http.Request) {
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

func (s *MusicServer) tracksAPI(w http.ResponseWriter, r *http.Request) {
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

func (s *MusicServer) tracksPage(w http.ResponseWriter, r *http.Request) {
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

func (s *MusicServer) coverImage(w http.ResponseWriter, r *http.Request) {
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

func (s *MusicServer) albumsAPI(w http.ResponseWriter, r *http.Request) {
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

func (s *MusicServer) albumAPI(w http.ResponseWriter, r *http.Request) {
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

func (s *MusicServer) albumsPage(w http.ResponseWriter, r *http.Request) {
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

func (s *MusicServer) downloadTrackAPI(w http.ResponseWriter, r *http.Request) {
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
