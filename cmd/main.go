package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"mime"
	"os"

	"github.com/bibrikthesorcerer/lantern/internal/config"
	"github.com/bibrikthesorcerer/lantern/internal/library"
	"github.com/bibrikthesorcerer/lantern/internal/web"
	clog "github.com/charmbracelet/log"
	"github.com/sawthorn/lantern/internal/config"
	"github.com/sawthorn/lantern/internal/library"
	"github.com/sawthorn/lantern/internal/web"
)

var musicDir string
var port int
var showConfig bool

func initParseCLIFlags() {
	flag.StringVar(&musicDir, "dir", "", "Path to music directory")
	flag.IntVar(&port, "port", 0, "Port on which server runs")
	flag.BoolVar(&showConfig, "show-Config", false, "Print current Config and exit")
	flag.Parse()
	mime.AddExtensionType(".css", "text/css; charset=utf-8")
	mime.AddExtensionType(".js", "application/javascript; charset=utf-8")
}

func flagsOverrideConfig(c *config.Config) {
	if musicDir != "" {
		c.Dir = musicDir
	}

	if port != 0 {
		c.Port = port
	}
}

func main() {
	initParseCLIFlags()

	conf, err := config.EnsureConfig()
	if err != nil {
		clog.Fatalf("can't ensure Config: %v", err)
	}

	if showConfig {
		data, _ := json.MarshalIndent(conf, "", " ")
		fmt.Println(string(data))
		os.Exit(0)
	}

	// override Config if needed
	flagsOverrideConfig(conf)

	// init cover cache
	coverCache, err := library.NewCoverCache()
	if err != nil {
		clog.Fatalf("cover cache init failed: %v", err)
	}

	// init library
	libraryRepo, err := library.NewLibraryRepository(conf.DBPath, coverCache)
	if err != nil {
		clog.Fatalf("library repo init failed: %v", err)
	}

	needsImport := libraryRepo.NeedsImport()
	if needsImport {
		clog.Info("library not present, starting full scan")
		if err := libraryRepo.ImportLibrary(conf.Dir); err != nil {
			clog.Fatalf("library import failed: %v", err)
		}
		clog.Info("full scan completed")
	} else {
		go func() {
			clog.Info("starting library sync")

			if err := libraryRepo.Sync(conf.Dir); err != nil {
				clog.Errorf("library sync failed: %v", err)
				return
			}

			clog.Info("library sync completed")
		}()
	}

	// http setup
	s, err := web.NewServer(libraryRepo, coverCache, conf, web.GetFS())
	if err != nil {
		clog.Fatalf("NewServer setup fail: %s", err)
	}

	web.PrintAddrQr(*s.Conf)

	clog.Fatal(s.RunServer()) // TODO: graceful shutdown
}
