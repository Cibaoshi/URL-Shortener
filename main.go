package main

import (
	"bufio"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math/big"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

type ClickInfo struct {
	Time time.Time `json:"time"`
	IP   string    `json:"ip"`
	UA   string    `json:"ua"`
}

type UrlStats struct {
	URL       string      `json:"url"`
	Clicks    int         `json:"clicks"`
	LastClick time.Time   `json:"last_click"`
	History   []ClickInfo `json:"history"`
}

type UrlStore struct {
	data map[string]*UrlStats
	mu   sync.RWMutex
}

var store = UrlStore{
	data: make(map[string]*UrlStats),
}

type ShortenRequest struct {
	URL string `json:"url"`
}

type ShortenResponse struct {
	ShortURL string `json:"short_url"`
}

type AppError struct {
	Code    int
	Title   string
	Message string
}

var (
	ErrNotFound = AppError{
		Code:    404,
		Title:   "404 — Not Found",
		Message: "Ссылка не существует или была удалена",
	}
	ErrMethod = AppError{
		Code:    405,
		Title:   "405 — Method Not Allowed",
		Message: "Используется неподдерживаемый HTTP метод",
	}
	ErrBadRequest = AppError{
		Code:    400,
		Title:   "400 — Bad Request",
		Message: "Некорректный запрос",
	}
)

var errorTpl = template.Must(template.New("error").Parse(`
<!DOCTYPE html>
<html lang="ru">
<head>
<meta charset="UTF-8">
<title>{{.Title}}</title>
<style>
body {
	background:#0f172a;
	color:#e5e7eb;
	font-family: monospace;
	display:flex;
	align-items:center;
	justify-content:center;
	height:100vh;
}
.box {
	border:1px solid #334155;
	padding:24px 32px;
	border-radius:8px;
	background:#020617;
}
h1 { color:#38bdf8; margin:0 0 8px; }
p  { margin:0; color:#cbd5f5; }
</style>
</head>
<body>
<div class="box">
	<h1>{{.Title}}</h1>
	<p>{{.Message}}</p>
</div>
</body>
</html>
`))

func main() {
	go func() {
		http.HandleFunc("/shorten", shortenHandler)
		http.HandleFunc("/", redirectHandler)

		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatal(err)
		}
	}()

	time.Sleep(100 * time.Millisecond)
	runConsoleUI()
}

func runConsoleUI() {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("\033[36m╔══════════════════════════════════════════════════════╗\033[0m")
	fmt.Println("\033[36m║\033[1;33m          URL SHORTENER — КОНСОЛЬНАЯ ПАНЕЛЬ           \033[0m\033[36m║\033[0m")
	fmt.Println("\033[36m╠══════════════════════════════════════════════════════╣\033[0m")
	fmt.Println("\033[36m║\033[0m Сервер: \033[32mhttp://localhost:8080\033[0m                        \033[36m║\033[0m")
	fmt.Println("\033[36m║\033[0m Команды: stat | stat <code> | top <n>                \033[36m║\033[0m")
	fmt.Println("\033[36m║\033[0m Для выхода нажмите: \033[31mCtrl+C\033[0m                           \033[36m║\033[0m")
	fmt.Println("\033[36m╚══════════════════════════════════════════════════════╝\033[0m")

	for {
		fmt.Print("\n\033[1mВведите URL или команду:\033[0m ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		parts := strings.Fields(input)

		switch parts[0] {
		case "stat":
			if len(parts) == 2 {
				showUrlStats(parts[1])
			} else {
				showStatistics()
			}
			continue
		case "top":
			if len(parts) == 2 {
				showTop(parts[1])
			}
			continue
		}

		code := saveUrl(input)
		fmt.Printf("\033[32mГотово: http://localhost:8080/%s\033[0m\n", code)
	}
}

func showStatistics() {
	store.mu.RLock()
	defer store.mu.RUnlock()

	if len(store.data) == 0 {
		fmt.Println("\033[31mСтатистика пуста\033[0m")
		return
	}

	fmt.Printf("\n%-8s | %-6s | %-16s | %s\n", "Код", "Клики", "Последний", "URL")
	fmt.Println("--------------------------------------------------------------")
	for code, s := range store.data {
		fmt.Printf("%-8s | %-6d | %-16s | %s\n",
			code, s.Clicks, formatTime(s.LastClick), s.URL)
	}
}

func showUrlStats(code string) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	s, ok := store.data[code]
	if !ok {
		fmt.Println("\033[31mКод не найден\033[0m")
		return
	}

	fmt.Printf("\nКод: %s\nURL: %s\nКлики: %d\n\n", code, s.URL, s.Clicks)
	for _, h := range s.History {
		fmt.Printf("%s | %-15s | %s\n",
			h.Time.Format("02.01 15:04"), h.IP, h.UA)
	}
}

func showTop(nStr string) {
	n := 0
	fmt.Sscanf(nStr, "%d", &n)
	if n <= 0 {
		return
	}

	type pair struct {
		Code   string
		Clicks int
	}

	store.mu.RLock()
	list := make([]pair, 0, len(store.data))
	for code, s := range store.data {
		list = append(list, pair{code, s.Clicks})
	}
	store.mu.RUnlock()

	sort.Slice(list, func(i, j int) bool {
		return list[i].Clicks > list[j].Clicks
	})

	if len(list) > n {
		list = list[:n]
	}

	for i, p := range list {
		fmt.Printf("%d. %s — %d\n", i+1, p.Code, p.Clicks)
	}
}

func saveUrl(originalURL string) string {
	for {
		code := generateShortCode()
		store.mu.Lock()
		if _, exists := store.data[code]; !exists {
			store.data[code] = &UrlStats{
				URL:     originalURL,
				History: make([]ClickInfo, 0),
			}
			store.mu.Unlock()
			return code
		}
		store.mu.Unlock()
	}
}

func generateShortCode() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 6)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[n.Int64()]
	}
	return string(b)
}

func shortenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		renderError(w, ErrMethod)
		return
	}

	var req ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		renderError(w, ErrBadRequest)
		return
	}

	code := saveUrl(req.URL)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ShortenResponse{
		ShortURL: "http://localhost:8080/" + code,
	})
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/favicon.ico" {
		return
	}

	code := strings.TrimPrefix(r.URL.Path, "/")
	ip := strings.Split(r.RemoteAddr, ":")[0]
	ua := r.UserAgent()

	store.mu.Lock()
	stats, ok := store.data[code]
	if ok {
		stats.Clicks++
		stats.LastClick = time.Now()
		stats.History = append(stats.History, ClickInfo{
			Time: stats.LastClick,
			IP:   ip,
			UA:   ua,
		})
		if len(stats.History) > 100 {
			stats.History = stats.History[len(stats.History)-100:]
		}
	}
	store.mu.Unlock()

	if !ok {
		renderError(w, ErrNotFound)
		return
	}

	http.Redirect(w, r, stats.URL, http.StatusFound)
}

func renderError(w http.ResponseWriter, err AppError) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(err.Code)
	_ = errorTpl.Execute(w, err)
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("02.01 15:04")
}
