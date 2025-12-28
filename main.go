package main

import (
	"bufio"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type UrlStore struct {
	urls map[string]string
	mu   sync.RWMutex
}

var store = UrlStore{
	urls: make(map[string]string),
}

type ShortenRequest struct {
	URL string `json:"url"`
}

type ShortenResponse struct {
	ShortURL string `json:"short_url"`
}

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
	fmt.Println("\033[36m║\033[0m Сервер активен на: \033[32mhttp://localhost:8080\033[0m             \033[36m║\033[0m")
	fmt.Println("\033[36m║\033[0m Логи переходов будут отображаться здесь               \033[36m║\033[0m") // Добавили инфо
	fmt.Println("\033[36m║\033[0m Для выхода нажмите: \033[31mCtrl+C\033[0m                           \033[36m║\033[0m")
	fmt.Println("\033[36m╚══════════════════════════════════════════════════════╝\033[0m")

	for {
		fmt.Print("\n\033[1mВведите URL:\033[0m ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		code := saveUrl(input)
		shortUrl := fmt.Sprintf("http://localhost:8080/%s", code)

		fmt.Println("\033[32m  └─ Готово!\033[0m")
		fmt.Printf("\033[1;33m     %s\033[0m\n", shortUrl)
	}
}

func saveUrl(originalURL string) string {
	code := generateShortCode()
	store.mu.Lock()
	store.urls[code] = originalURL
	store.mu.Unlock()
	return code
}

func generateShortCode() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, 6)
	for i := range code {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		code[i] = charset[num.Int64()]
	}
	return string(code)
}

func shortenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	var req ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Неверный формат JSON", http.StatusBadRequest)
		return
	}

	code := saveUrl(req.URL)

	resp := ShortenResponse{
		ShortURL: fmt.Sprintf("http://localhost:8080/%s", code),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/favicon.ico" {
		return
	}

	code := r.URL.Path[1:]

	store.mu.RLock()
	originalURL, ok := store.urls[code]
	store.mu.RUnlock()

	if !ok {
		// Лог ошибки (красный)
		fmt.Printf("\n\033[31m[LOG %s] Ошибка: код %s не найден\033[0m\n", time.Now().Format("15:04:05"), code)
		http.NotFound(w, r)
		return
	}

	// Лог успешного перехода (синий)
	fmt.Printf("\n\033[34m[LOG %s] Переход по коду %s -> %s\033[0m\n", time.Now().Format("15:04:05"), code, originalURL)

	http.Redirect(w, r, originalURL, http.StatusFound)
}
