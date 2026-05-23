package site

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"HLTV-Manager/logger"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func (site *Site) terminalHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/admin/hltv/terminal/"):]
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

	if websocket.IsWebSocketUpgrade(r) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.ErrorLogger.Printf("WebSocket upgrade error for HLTV %d: %v", id, err)
			return
		}
		defer conn.Close()

		ch := make(chan string, 50)
		h.Subscribe(ch)
		defer h.Unsubscribe(ch)

		for _, line := range h.GetLogSnapshot() {
			if err := conn.WriteMessage(websocket.TextMessage, []byte(line)); err != nil {
				return
			}
		}

		// Приём команд от клиента
		done := make(chan struct{})
		go func() {
			defer close(done)
			for {
				_, msg, err := conn.ReadMessage()
				if err != nil {
					return
				}
				cmd := strings.TrimSpace(string(msg))
				if cmd != "" {
					if err := h.WriteCommand(cmd); err != nil {
						fmt.Printf("Write command error: %v\n", err)
					}
				}
			}
		}()

		for {
			select {
			case line, ok := <-ch:
				if !ok {
					return
				}
				if err := conn.WriteMessage(websocket.TextMessage, []byte(line)); err != nil {
					return
				}
			case <-done:
				return
			}
		}
	}

	tmpl, err := template.ParseFiles(
		filepath.Join("frontend", "head.gohtml"),
		filepath.Join("frontend", "navbar.gohtml"),
		filepath.Join("frontend", "admin", "terminal.gohtml"),
	)
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "terminal", h)
}