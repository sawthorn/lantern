package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"

	clog "github.com/charmbracelet/log"
	"github.com/sawthorn/lantern/internal/library/repository"
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
	router.HandleFunc("GET /albums/{id}", s.albumPage)
	router.HandleFunc("GET /api/stream/{id}", s.streamTrackAPI)
	router.HandleFunc("GET /api/albums/{id}", s.albumAPI)
	router.HandleFunc("GET /api/tracks", s.tracksAPI)
	router.HandleFunc("GET /api/covers/{id}", s.coverImage)
	router.HandleFunc("GET /api/albums", s.albumsAPI)
	router.HandleFunc("GET /api/tracks/{id}/download", s.downloadTrackAPI)
	router.HandleFunc("GET /api/search", s.searchAPI)

	return router
}

func (s *MusicServer) streamTrackAPI(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		clog.Errorf("stream track: %s", err)
		http.Error(w, "ID must be an integer", http.StatusBadRequest)
		return
	}

	track, err := s.repo.GetTrackByID(r.Context(), uint16(id))
	if errors.Is(err, repository.ErrTrackNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		clog.Errorf("stream err for track #%d. %s", id, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	f, err := os.Open(track.FSInfo.Path)
	if err != nil {
		clog.Errorf("file %q open fail: %s", track.FSInfo.Path, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
	defer f.Close()
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeContent(w, r, track.FSInfo.Filename, track.FSInfo.ModTime, f)
}

func (s *MusicServer) tracksAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	tracks, err := s.repo.GetAllTracks(r.Context())
	if err != nil {
		clog.Errorf("get album list json: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(tracks)
	if err != nil {
		clog.Errorf("json encode fail: %s", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func (s *MusicServer) tracksPage(w http.ResponseWriter, r *http.Request) {
	tracks, err := s.repo.GetAllTracks(r.Context())
	if err != nil {
		clog.Errorf("get track list json: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	err = s.views.Render(w, "tracks", tracks)
	if err != nil {
		clog.Errorf("render tracks: %s", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

func (s *MusicServer) coverImage(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		clog.Errorf("id conversion fail: %s", err)
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

	albums, err := s.repo.GetAlbums(r.Context())
	if err != nil {
		clog.Errorf("get album list json: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(albums)
	if err != nil {
		clog.Errorf("json encode fail: %s", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func (s *MusicServer) albumAPI(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		clog.Errorf("incorrect id: %s", err)
		http.Error(w, "ID must be an integer", http.StatusBadRequest)
		return
	}

	albums, err := s.repo.GetAlbumByID(r.Context(), uint16(id))
	if errors.Is(err, repository.ErrAlbumNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		clog.Errorf("album retrieve fail: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(albums)
	if err != nil {
		clog.Errorf("json encode fail: %s", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func (s *MusicServer) albumsPage(w http.ResponseWriter, r *http.Request) {
	albums, err := s.repo.GetAlbums(r.Context())
	if err != nil {
		clog.Errorf("get albums from repo fail: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	err = s.views.Render(w, "albums", albums)
	if err != nil {
		clog.Errorf("render albums: %s", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

func (s *MusicServer) albumPage(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		clog.Errorf("album ID conversion: %s", err)
		http.Error(w, "ID must be an integer", http.StatusBadRequest)
		return
	}

	album, err := s.repo.GetAlbumByID(r.Context(), uint16(id))
	if errors.Is(err, repository.ErrAlbumNotFound) {
		http.NotFound(w, r)
		return
	}

	err = s.views.Render(w, "album", album)
	if err != nil {
		clog.Errorf("render album: %s", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

func (s *MusicServer) downloadTrackAPI(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		clog.Errorf("incorrect id: %s", err)
		http.Error(w, "ID must be an integer", http.StatusBadRequest)
		return
	}

	track, err := s.repo.GetTrackByID(r.Context(), uint16(id))
	if err != nil {
		clog.Errorf("get track: %s", err)
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Disposition", "attachment; filename="+track.FSInfo.Path)
	w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
	http.ServeFile(w, r, track.FSInfo.Path)
}

func (s *MusicServer) searchAPI(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		http.Error(w, "query parameter q is required", http.StatusBadRequest)
		return
	}

	results, err := s.repo.Search(r.Context(), query)
	if err != nil {
		clog.Errorf("search library: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	body, err := json.Marshal(results)
	if err != nil {
		clog.Errorf("encode search results: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(body); err != nil {
		clog.Errorf("write search response: %v", err)
	}
}
