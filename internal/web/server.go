package web

import (
	"fmt"
	"io/fs"
	"net/http"

	"github.com/sawthorn/lantern/internal/config"
	"github.com/sawthorn/lantern/internal/library"
	"github.com/sawthorn/lantern/internal/web/middleware"

	clog "github.com/charmbracelet/log"
)

type MusicServer struct {
	repo        *library.LibraryRepository
	coverCache  *library.CoverCache
	Conf        *config.Config
	staticFiles fs.FS
	views       *Views
}

func NewServer(
	repo *library.LibraryRepository,
	coverCache *library.CoverCache,
	c *config.Config,
	staticFiles fs.FS,
	views *Views,
) *MusicServer {
	return &MusicServer{
		repo:        repo,
		coverCache:  coverCache,
		Conf:        c,
		staticFiles: staticFiles,
		views:       views,
	}
}

func (s *MusicServer) RunServer() error {
	router := SetUpRouting(s)

	addrString := fmt.Sprintf(":%d", s.Conf.Port)
	srv := &http.Server{
		Addr:    addrString,
		Handler: middleware.Logging(router),
	}

	clog.Infof("Launching server on %s", addrString)
	return fmt.Errorf("server shutdown: %w", srv.ListenAndServe())
}
