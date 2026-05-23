package site

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"HLTV-Manager/config"
)

// apiAuth проверяет токен
func apiAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Api-Token")
		if token == "" {
			token = r.URL.Query().Get("token")
		}
		if token != config.ApiToken() || token == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// GET /api/hltv
func (site *Site) apiListHLTV(w http.ResponseWriter, r *http.Request) {
	type hltvInfo struct {
		ID        int    `json:"id"`
		Name      string `json:"name"`
		ShowIP    string `json:"show_ip"`
		Connect   string `json:"connect"`
		Port      string `json:"port"`
		IsRunning bool   `json:"is_running"`
	}
	list := make([]hltvInfo, 0, len(site.HLTV))
	for _, h := range site.HLTV {
		list = append(list, hltvInfo{
			ID:        h.ID,
			Name:      h.Settings.Name,
			ShowIP:    h.Settings.ShowIP,
			Connect:   h.Settings.Connect,
			Port:      h.Settings.Port,
			IsRunning: h.IsRunning(),
		})
	}
	writeJSON(w, list)
}

// GET /api/hltv/{id}
func (site *Site) apiGetHLTV(w http.ResponseWriter, r *http.Request) {
	id, err := extractID(r, "/api/hltv/")
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}
	h, err := site.getHLTVByID(id)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	writeJSON(w, map[string]interface{}{
		"id":           h.ID,
		"name":         h.Settings.Name,
		"show_ip":      h.Settings.ShowIP,
		"connect":      h.Settings.Connect,
		"port":         h.Settings.Port,
		"is_running":   h.IsRunning(),
		"max_demo_day": h.Settings.MaxDemoDay,
	})
}

// POST /api/hltv/{id}/start
func (site *Site) apiStartHLTV(w http.ResponseWriter, r *http.Request) {
	id, err := extractID(r, "/api/hltv/")
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}
	h, err := site.getHLTVByID(id)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	if h.IsRunning() {
		writeJSON(w, map[string]string{"status": "already running"})
		return
	}
	if err := h.Start(); err != nil {
		writeJSON(w, map[string]string{"status": "error", "message": err.Error()})
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

// POST /api/hltv/{id}/stop
func (site *Site) apiStopHLTV(w http.ResponseWriter, r *http.Request) {
	id, err := extractID(r, "/api/hltv/")
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}
	h, err := site.getHLTVByID(id)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	if !h.IsRunning() {
		writeJSON(w, map[string]string{"status": "already stopped"})
		return
	}
	if err := h.Quit(); err != nil {
		writeJSON(w, map[string]string{"status": "error", "message": err.Error()})
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

// GET /api/hltv/{id}/demos
func (site *Site) apiGetDemos(w http.ResponseWriter, r *http.Request) {
	id, err := extractID(r, "/api/hltv/")
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}
	h, err := site.getHLTVByID(id)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	h.DemoControl()
	demos := make([]map[string]interface{}, 0, len(h.Demos))
	for _, d := range h.Demos {
		demos = append(demos, map[string]interface{}{
			"id":   d.ID,
			"name": d.Name,
			"date": d.Date,
			"time": d.Time,
			"map":  d.Map,
		})
	}
	writeJSON(w, demos)
}

// GET /api/hltv/{id}/log
func (site *Site) apiGetLog(w http.ResponseWriter, r *http.Request) {
	id, err := extractID(r, "/api/hltv/")
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}
	h, err := site.getHLTVByID(id)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	lines := h.GetLogSnapshot()
	if lines == nil {
		lines = []string{}
	}
	writeJSON(w, lines)
}

// POST /api/hltv/{id}/record/start
func (site *Site) recordStartHandler(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Api-Token")
	if token == "" {
		token = r.URL.Query().Get("token")
	}
	if token != config.ApiToken() || token == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/api/hltv/")
	idStr = strings.TrimSuffix(idStr, "/record/start")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	h, err := site.getHLTVByID(id)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}

	if !h.IsRunning() {
		writeJSON(w, map[string]string{"status": "error", "message": "HLTV is not running"})
		return
	}

	demoName := r.FormValue("demo_name")
	if demoName == "" {
		demoName = h.Settings.DemoName
	}

	cmd := "record " + demoName + "\n"
	if err := h.WriteCommand(cmd); err != nil {
		writeJSON(w, map[string]string{"status": "error", "message": err.Error()})
		return
	}

	writeJSON(w, map[string]string{"status": "ok", "demo_name": demoName})
}

// POST /api/hltv/{id}/record/stop
func (site *Site) recordStopHandler(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Api-Token")
	if token == "" {
		token = r.URL.Query().Get("token")
	}
	if token != config.ApiToken() || token == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/api/hltv/")
	idStr = strings.TrimSuffix(idStr, "/record/stop")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	h, err := site.getHLTVByID(id)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}

	if !h.IsRunning() {
		writeJSON(w, map[string]string{"status": "error", "message": "HLTV is not running"})
		return
	}

	if err := h.WriteCommand("stoprecording\n"); err != nil {
		writeJSON(w, map[string]string{"status": "error", "message": err.Error()})
		return
	}

	writeJSON(w, map[string]string{"status": "ok"})
}

// Вспомогательные функции

func extractID(r *http.Request, prefix string) (int, error) {
	sub := r.URL.Path[len(prefix):]
	for i, c := range sub {
		if c == '/' {
			sub = sub[:i]
			break
		}
	}
	return strconv.Atoi(sub)
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}