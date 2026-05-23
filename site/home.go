package site

import (
	"net/http"
	"path/filepath"
	"text/template"
)

func (site *Site) homeHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles(
		filepath.Join("frontend", "head.gohtml"),
		filepath.Join("frontend", "navbar.gohtml"),
		filepath.Join("frontend", "home.gohtml"),
	)

	if err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.ExecuteTemplate(w, "home", site.HLTV)
	if err != nil {
		http.Error(w, "Rendering error: "+err.Error(), http.StatusInternalServerError)
	}
}
