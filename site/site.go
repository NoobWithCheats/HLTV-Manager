package site

import (
	"HLTV-Manager/hltv"
	"net/http"
	"strings"
	"strconv"
)

type Site struct {
	HLTV []*hltv.HLTV
}

func (site *Site) Init() {
	// Статические файлы
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("frontend"))))

	// Публичные маршруты
	http.HandleFunc("/", site.homeHandler)
	http.HandleFunc("/demos/", site.demosHandler)
	http.HandleFunc("/download/", site.downloadHandler)

	// Админка
	http.HandleFunc("/admin", authMiddleware(site.adminDashboard))
	http.HandleFunc("/admin/login", site.loginHandler)
	http.HandleFunc("/admin/logout", site.logoutHandler)
	http.HandleFunc("/admin/change-password", authMiddleware(site.changePasswordHandler))
	http.HandleFunc("/admin/hltv/new", authMiddleware(site.newHLTVHandler))
	http.HandleFunc("/admin/hltv/edit/", authMiddleware(site.editHLTVHandler))
	http.HandleFunc("/admin/hltv/delete/", authMiddleware(site.deleteHLTVHandler))
	http.HandleFunc("/admin/hltv/start/", authMiddleware(site.startHLTVHandler))
	http.HandleFunc("/admin/hltv/stop/", authMiddleware(site.stopHLTVHandler))
	http.HandleFunc("/admin/hltv/terminal/", authMiddleware(site.terminalHandler))

	// API маршруты (все в одном обработчике)
	http.HandleFunc("/api/hltv", apiAuth(site.apiListHLTV))
	http.HandleFunc("/api/hltv/", apiAuth(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case path == "/api/hltv":
			site.apiListHLTV(w, r)
		case isSimpleID(path):
			site.apiGetHLTV(w, r)
		case strings.HasSuffix(path, "/start"):
			site.apiStartHLTV(w, r)
		case strings.HasSuffix(path, "/stop"):
			site.apiStopHLTV(w, r)
		case strings.HasSuffix(path, "/demos"):
			site.apiGetDemos(w, r)
		case strings.HasSuffix(path, "/log"):
			site.apiGetLog(w, r)
		case strings.HasSuffix(path, "/record/start"):
			site.recordStartHandler(w, r)
		case strings.HasSuffix(path, "/record/stop"):
			site.recordStopHandler(w, r)
		default:
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		}
	}))
}

func isSimpleID(path string) bool {
	parts := strings.SplitN(path, "/", 4)
	if len(parts) == 3 {
		_, err := strconv.Atoi(parts[2])
		return err == nil
	}
	return false
}