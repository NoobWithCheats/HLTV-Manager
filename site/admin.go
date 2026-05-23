package site

import (
    "HLTV-Manager/config"
    "HLTV-Manager/hltv"
    "HLTV-Manager/reader"
    "fmt"
    "net/http"
    "os"
    "strconv"
    "strings"
    "text/template"

    "gopkg.in/yaml.v3"
)

func (site *Site) adminDashboard(w http.ResponseWriter, r *http.Request) {
    tmpl := template.Must(template.ParseFiles(
        "frontend/head.gohtml",
        "frontend/navbar.gohtml",
        "frontend/admin/dashboard.gohtml",
    ))
    tmpl.ExecuteTemplate(w, "dashboard", site.HLTV)
}


func (site *Site) newHLTVHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method == http.MethodPost {
        settings := hltv.Settings{
            Name:       r.FormValue("name"),
            ShowIP:     r.FormValue("showip"),
            Connect:    r.FormValue("connect"),
            Port:       r.FormValue("port"),
            GameID:     r.FormValue("gameid"),
            DemoName:   r.FormValue("demoname"),
            MaxDemoDay: r.FormValue("maxdemoday"),
            DebugTerminalLog: r.FormValue("debug") == "on",
            Cvars:      parseCvars(r.FormValue("cvars")),
        }

        newHLTV, err := hltv.NewHLTV(site.nextID(), settings)
        if err != nil {
            http.Error(w, "Ошибка создания HLTV: "+err.Error(), http.StatusInternalServerError)
            return
        }
        site.HLTV = append(site.HLTV, newHLTV)

        if err := newHLTV.Start(); err != nil {
            http.Error(w, "Ошибка старта HLTV: "+err.Error(), http.StatusInternalServerError)
            return
        }

        if err := site.saveRunnersConfig(); err != nil {
            http.Error(w, "HLTV запущен, но не удалось сохранить конфигурацию: "+err.Error(), http.StatusInternalServerError)
            return
        }

        http.Redirect(w, r, "/admin", http.StatusSeeOther)
        return
    }

    tmpl := template.Must(template.ParseFiles(
        "frontend/head.gohtml",
        "frontend/navbar.gohtml",
        "frontend/admin/hltv_form.gohtml",
    ))
    tmpl.ExecuteTemplate(w, "hltv_form", nil)
}

func (site *Site) editHLTVHandler(w http.ResponseWriter, r *http.Request) {
    idStr := r.URL.Path[len("/admin/hltv/edit/"):]
    id, err := strconv.Atoi(idStr)
    if err != nil {
        http.Error(w, "Invalid ID", http.StatusBadRequest)
        return
    }

    h, err := site.getHLTVByID(id)
    if err != nil {
        http.Error(w, "HLTV not found", http.StatusNotFound)
        return
    }

    if r.Method == http.MethodPost {
        h.Settings = hltv.Settings{
            Name:       r.FormValue("name"),
            ShowIP:     r.FormValue("showip"),
            Connect:    r.FormValue("connect"),
            Port:       r.FormValue("port"),
            GameID:     r.FormValue("gameid"),
            DemoName:   r.FormValue("demoname"),
            MaxDemoDay: r.FormValue("maxdemoday"),
            DebugTerminalLog: r.FormValue("debug") == "on",
            Cvars:      parseCvars(r.FormValue("cvars")),
        }

        if err := h.Restart(); err != nil {
            http.Error(w, "Ошибка перезапуска: "+err.Error(), http.StatusInternalServerError)
            return
        }

        if err := site.saveRunnersConfig(); err != nil {
            http.Error(w, "Конфигурация обновлена, но не сохранена: "+err.Error(), http.StatusInternalServerError)
            return
        }

        http.Redirect(w, r, "/admin", http.StatusSeeOther)
        return
    }

    tmpl := template.Must(template.ParseFiles(
        "frontend/head.gohtml",
        "frontend/navbar.gohtml",
        "frontend/admin/hltv_form.gohtml",
    ))
    tmpl.ExecuteTemplate(w, "hltv_form", h)
}

func (site *Site) deleteHLTVHandler(w http.ResponseWriter, r *http.Request) {
    idStr := r.URL.Path[len("/admin/hltv/delete/"):]
    id, err := strconv.Atoi(idStr)
    if err != nil {
        http.Error(w, "Invalid ID", http.StatusBadRequest)
        return
    }

    h, err := site.getHLTVByID(id)
    if err != nil {
        http.Error(w, "HLTV not found", http.StatusNotFound)
        return
    }

    h.Quit()

    newList := make([]*hltv.HLTV, 0)
    for _, hl := range site.HLTV {
        if hl.ID != id {
            newList = append(newList, hl)
        }
    }
    site.HLTV = newList

    if err := site.saveRunnersConfig(); err != nil {
        http.Error(w, "HLTV удалён, но конфигурация не сохранена: "+err.Error(), http.StatusInternalServerError)
        return
    }

    http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (site *Site) startHLTVHandler(w http.ResponseWriter, r *http.Request) {
    idStr := r.URL.Path[len("/admin/hltv/start/"):]
    id, _ := strconv.Atoi(idStr)
    h, err := site.getHLTVByID(id)
    if err != nil {
        http.Error(w, "HLTV not found", http.StatusNotFound)
        return
    }
    if err := h.Start(); err != nil {
        http.Error(w, "Ошибка старта: "+err.Error(), http.StatusInternalServerError)
        return
    }
    http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (site *Site) stopHLTVHandler(w http.ResponseWriter, r *http.Request) {
    idStr := r.URL.Path[len("/admin/hltv/stop/"):]
    id, _ := strconv.Atoi(idStr)
    h, err := site.getHLTVByID(id)
    if err != nil {
        http.Error(w, "HLTV not found", http.StatusNotFound)
        return
    }
    h.Quit()
    http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (site *Site) nextID() int {
    max := 0
    for _, h := range site.HLTV {
        if h.ID > max {
            max = h.ID
        }
    }
    return max + 1
}

func (site *Site) getHLTVByID(id int) (*hltv.HLTV, error) {
    for _, h := range site.HLTV {
        if h.ID == id {
            return h, nil
        }
    }
    return nil, fmt.Errorf("не найден")
}

func parseCvars(text string) []string {
    lines := strings.Split(text, "\n")
    var result []string
    for _, line := range lines {
        line = strings.TrimSpace(line)
        if line != "" && !strings.HasPrefix(line, "#") {
            result = append(result, line)
        }
    }
    return result
}

func (site *Site) saveRunnersConfig() error {
    configPath := config.HltvRunnerFile()
    runners := make([]reader.HLTV, 0, len(site.HLTV))
    for _, h := range site.HLTV {
        r := reader.HLTV{
            Name:             h.Settings.Name,
            ShowIP:           h.Settings.ShowIP,
            Connect:          h.Settings.Connect,
            Port:             h.Settings.Port,
            GameID:           h.Settings.GameID,
            DemoName:         h.Settings.DemoName,
            MaxDemoDay:       h.Settings.MaxDemoDay,
            DebugTerminalLog: h.Settings.DebugTerminalLog,
            Cvars:            h.Settings.Cvars,
        }
        runners = append(runners, r)
    }

    cfg := reader.Config{HLTV: runners}
    data, err := yaml.Marshal(&cfg)
    if err != nil {
        return err
    }
    return os.WriteFile(configPath, data, 0644)
}