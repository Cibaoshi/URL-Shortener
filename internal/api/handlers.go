package api

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strings"

	"shortenergo/internal/storage"
	"shortenergo/internal/utils"
)

type Handler struct {
	Store *storage.Store
}

func NewHandler(store *storage.Store) *Handler {
	return &Handler{Store: store}
}

type ShortenRequest struct {
	URL string `json:"url"`
}

type ShortenResponse struct {
	ShortURL string `json:"short_url"`
}

func (h *Handler) ShortenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.renderError(w, 405, "Method Not Allowed", "Используйте POST запрос")
		return
	}

	var req ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.renderError(w, 400, "Bad Request", "Некорректный JSON")
		return
	}

	cleanURL, err := utils.ValidateURL(req.URL)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	code := h.Store.Save(cleanURL)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ShortenResponse{
		ShortURL: "http://localhost:8080/" + code,
	})
}

func (h *Handler) RedirectHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/favicon.ico" {
		return
	}

	code := strings.TrimPrefix(r.URL.Path, "/")

	if code == "" {
		w.Write([]byte("Shortener API is running"))
		return
	}

	stats, ok := h.Store.Get(code)
	if !ok {
		h.renderError(w, 404, "Not Found", "Ссылка не найдена")
		return
	}

	// Асинхронно или быстро обновляем статистику
	ip := strings.Split(r.RemoteAddr, ":")[0]
	h.Store.AddClick(code, ip, r.UserAgent())

	http.Redirect(w, r, stats.URL, http.StatusFound)
}

func (h *Handler) renderError(w http.ResponseWriter, code int, title, message string) {
	tpl := template.Must(template.New("error").Parse(`
<!DOCTYPE html>
<html lang="ru">
<head><meta charset="UTF-8"><title>{{.Title}}</title>
<style>body{background:#0f172a;color:#e5e7eb;font-family:monospace;display:flex;align-items:center;justify-content:center;height:100vh}
.box{border:1px solid #334155;padding:24px 32px;border-radius:8px;background:#020617}h1{color:#38bdf8;margin:0 0 8px}</style>
</head><body><div class="box"><h1>{{.Title}}</h1><p>{{.Message}}</p></div></body></html>`))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	_ = tpl.Execute(w, map[string]string{"Title": title, "Message": message})
}
