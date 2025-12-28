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

// UrlStore хранит маппинг ссылок
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
	// Запуск HTTP сервера в фоновом режиме
	go func() {
		http.HandleFunc("/shorten", shortenHandler)
		http.HandleFunc("/", redirectHandler)

		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatal(err)
		}
	}()

	// Небольшая пауза для инициализации сервера
	time.Sleep(100 * time.Millisecond)

	// Запуск консольного интерфейса
	runConsoleUI()
}

func runConsoleUI() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("--- Сервер запущен на :8080 ---")
	fmt.Println("Введите ссылку и нажмите Enter для сокращения (или Ctrl+C для выхода):")

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		code := saveUrl(input)
		fmt.Printf("Короткая ссылка: http://localhost:8080/%s\n\n", code)
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
		http.NotFound(w, r)
		return
	}

	http.Redirect(w, r, originalURL, http.StatusFound)
}
