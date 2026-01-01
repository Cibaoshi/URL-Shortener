package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

// --- Структуры данных ---

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

// --- Ошибки и шаблоны ---

var (
	ErrNotFound   = AppError{404, "404 — Not Found", "Ссылка не существует или была удалена"}
	ErrMethod     = AppError{405, "405 — Method Not Allowed", "Используется неподдерживаемый HTTP метод"}
	ErrBadRequest = AppError{400, "400 — Bad Request", "Некорректный запрос"}
)

var errorTpl = template.Must(template.New("error").Parse(`
<!DOCTYPE html>
<html lang="ru">
<head>
<meta charset="UTF-8">
<title>{{.Title}}</title>
<style>
body{background:#0f172a;color:#e5e7eb;font-family:monospace;display:flex;align-items:center;justify-content:center;height:100vh}
.box{border:1px solid #334155;padding:24px 32px;border-radius:8px;background:#020617}
h1{color:#38bdf8;margin:0 0 8px}
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

// --- Main ---

func main() {
	// 🔹 серверные логи → stderr
	log.SetOutput(os.Stderr)
	log.SetFlags(log.Ltime)

	mux := http.NewServeMux()
	mux.HandleFunc("/shorten", shortenHandler)
	mux.HandleFunc("/", redirectHandler)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		log.Println("server started on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	go runConsoleUI()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)

	log.Println("server stopped")
}

// --- UI и Логика консоли ---

func runConsoleUI() {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("\033[36m╔══════════════════════════════════════════════════════╗\033[0m")
	fmt.Println("\033[36m║\033[1;33m          URL SHORTENER — КОНСОЛЬ                     \033[0m\033[36m║\033[0m")
	fmt.Println("\033[36m╠══════════════════════════════════════════════════════╣\033[0m")
	fmt.Println("\033[36m║\033[0m Сервер: \033[32mhttp://localhost:8080\033[0m                        \033[36m║\033[0m")
	fmt.Println("\033[36m║\033[0m Команды: stat | stat <code> | top <n>                \033[36m║\033[0m")
	fmt.Println("\033[36m╚══════════════════════════════════════════════════════╝\033[0m")

	for {
		fmt.Print("\n\033[1mВведите URL или команду, запуск сервера :\033[0m ")
		if !scanner.Scan() {
			return
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

		// Валидация введенного в консоль URL
		cleanURL, err := validateURL(input)
		if err != nil {
			fmt.Printf("\033[31mОшибка валидации: %v\033[0m\n", err)
			continue
		}

		code := saveUrl(cleanURL)
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

// --- Бизнес-логика ---

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

// validateURL проверяет и нормализует URL
func validateURL(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)

	if rawURL == "" {
		return "", fmt.Errorf("пустой URL")
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("некорректный формат")
	}

	// Если пользователь ввел "google.com" без http
	if u.Scheme == "" && u.Host == "" {
		// Проверяем, есть ли точка (признак домена), иначе это просто мусор
		if !strings.Contains(rawURL, ".") {
			return "", fmt.Errorf("это не похоже на адрес сайта (нет точки)")
		}
		rawURL = "https://" + rawURL
		u, _ = url.Parse(rawURL) // перепарсиваем с протоколом
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("поддерживается только http и https")
	}

	if u.Host == "" {
		return "", fmt.Errorf("отсутствует домен")
	}

	// Проверка на наличие точки в хосте (защита от "http://test")
	if !strings.Contains(u.Host, ".") && u.Host != "localhost" {
		return "", fmt.Errorf("некорректное доменное имя")
	}

	if u.Host == "localhost:8080" {
		return "", fmt.Errorf("нельзя сокращать ссылки этого сервиса")
	}

	return rawURL, nil
}

// --- HTTP Handlers ---

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

	// Валидация
	cleanURL, err := validateURL(req.URL)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	code := saveUrl(cleanURL)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
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
