package site

import (
	"HLTV-Manager/hltv"
	"net/http"
	"path/filepath"
	"strconv"
	"text/template"
)

func (site *Site) demosHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/demos/"):]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid HLTV ID.", http.StatusBadRequest)
		return
	}

	var hltv *hltv.HLTV
	for _, h := range site.HLTV {
		if h.ID == id {
			hltv = h
			break
		}
	}

	if hltv == nil {
		http.Error(w, "HLTV server not found.", http.StatusNotFound)
		return
	}

	tmpl, err := template.ParseFiles(
		filepath.Join("frontend", "head.gohtml"),
		filepath.Join("frontend", "navbar.gohtml"),
		filepath.Join("frontend", "demos.gohtml"),
	)

	if err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.ExecuteTemplate(w, "demos", hltv)
	if err != nil {
		http.Error(w, "Rendering error: "+err.Error(), http.StatusInternalServerError)
	}
}
