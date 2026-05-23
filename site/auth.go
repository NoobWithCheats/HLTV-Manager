package site

import (
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const sessionCookieName = "hltv_session"

var (
	adminUser      string
	adminPassHash  string
	hashFilePath   string
)

func init() {
	adminUser = os.Getenv("ADMIN_USER")
	if adminUser == "" {
		adminUser = "admin"
	}

	hashFilePath = filepath.Join("/app", "log", "admin_pass_hash")
	loadAdminHash()
}

func loadAdminHash() {
	data, err := os.ReadFile(hashFilePath)
	if err == nil && len(data) > 0 {
		adminPassHash = string(data)
		return
	}

	adminPassHash = os.Getenv("ADMIN_PASS_HASH")
}

func saveAdminHash(hash string) error {
	return os.WriteFile(hashFilePath, []byte(hash), 0600)
}

func verifyPassword(password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(adminPassHash), []byte(password)) == nil
}

func (site *Site) changePasswordHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{}
	if r.Method == http.MethodPost {
		current := r.FormValue("current_password")
		newPass := r.FormValue("new_password")
		confirm := r.FormValue("confirm_password")

		if !verifyPassword(current) {
			data["Error"] = "Неверный текущий пароль"
		} else if newPass != confirm {
			data["Error"] = "Пароли не совпадают"
		} else if len(newPass) < 4 {
			data["Error"] = "Новый пароль должен быть не менее 4 символов"
		} else {
			hash, err := bcrypt.GenerateFromPassword([]byte(newPass), bcrypt.DefaultCost)
			if err != nil {
				data["Error"] = "Ошибка хеширования пароля"
			} else {
				adminPassHash = string(hash)
				if err := saveAdminHash(adminPassHash); err != nil {
					data["Error"] = "Не удалось сохранить пароль (ошибка записи файла)"
				} else {
					data["Success"] = "Пароль успешно изменён"
				}
			}
		}
	}

	tmpl := template.Must(template.ParseFiles(
		"frontend/head.gohtml",
		"frontend/navbar.gohtml",
		"frontend/admin/change_password.gohtml",
	))
	tmpl.ExecuteTemplate(w, "change_password", data)
}

func (site *Site) loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")
		if username == adminUser && verifyPassword(password) {
			http.SetCookie(w, &http.Cookie{
				Name:    sessionCookieName,
				Value:   "authenticated",
				Path:    "/",
				Expires: time.Now().Add(24 * time.Hour),
			})
			http.Redirect(w, r, "/admin", http.StatusSeeOther)
			return
		}
		http.Error(w, "Неверный логин или пароль", http.StatusUnauthorized)
		return
	}
	tmpl, _ := template.ParseFiles(
		"frontend/head.gohtml",
		"frontend/navbar.gohtml",
		"frontend/admin/login.gohtml",
	)
	tmpl.ExecuteTemplate(w, "login", nil)
}

func (site *Site) logoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:    sessionCookieName,
		Value:   "",
		Path:    "/",
		Expires: time.Now().Add(-1 * time.Hour),
	})
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil || cookie.Value != "authenticated" {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}