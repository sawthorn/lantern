package web

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
)

// Views renders application's HTML templates
type Views struct {
	pages map[string]*template.Template
}

// NewViews parses HTML templates from `templateFS` and returns a Views renderer.
func NewViews(templateFS fs.FS) (*Views, error) {
	base, err := template.ParseFS(
		templateFS,
		"base.html",
		"nav.html",
		"player.html",
		"components/*.html",
	)
	if err != nil {
		return nil, fmt.Errorf("base templates parse: %w", err)
	}

	pages := make(map[string]*template.Template)

	pagesFiles := map[string]string{
		"home":   "home.html",
		"albums": "albums_page.html",
		"album":  "album_page.html",
		"tracks": "tracks_page.html",
	}

	for name, path := range pagesFiles {
		t, err := base.Clone()
		if err != nil {
			return nil, fmt.Errorf("clone templates for %q: %w", name, err)
		}

		if _, err := t.ParseFS(templateFS, path); err != nil {
			return nil, fmt.Errorf("parse page %q: %w", name, err)
		}

		pages[name] = t
	}

	return &Views{pages: pages}, nil
}

// Render executes the view named `page` with `data` and writes the result to `w`.
// It buffers the rendered output so template execution errors do not partially
// write the HTTP response.
func (v *Views) Render(w http.ResponseWriter, page string, data any) error {
	t, ok := v.pages[page]
	if !ok {
		return fmt.Errorf("unknown view %q", page)
	}

	var buf bytes.Buffer

	if err := t.ExecuteTemplate(&buf, "base", data); err != nil {
		return fmt.Errorf("execute view %q: %w", page, err)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := buf.WriteTo(w); err != nil {
		return fmt.Errorf("write view %q: %w", page, err)
	}
	return nil
}
