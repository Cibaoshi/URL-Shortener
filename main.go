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
	urls   map[string]string
	clicks map[string]int // Добавили хранилище для кликов
	mu     sync.RWMutex
}

var store = UrlStore{
	urls:   make(map[string]string),
	clicks: make(map[string]int), // Инициализация счетчика
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
	fmt.Println("\033[36m║\033[0m Сервер: \033[32mhttp://localhost:8080\033[0m                        \033[36m║\033[0m")
	fmt.Println("\033[36m║\033[0m Введите \033[1;35mstat\033[0m для просмотра статистики                \033[36m║\033[0m")
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

		// Обработка команды статистики
		if input == "stat" {
			showStatistics()
			continue
		}

		code := saveUrl(input)
		shortUrl := fmt.Sprintf("http://localhost:8080/%s", code)

		fmt.Println("\033[32m  └─ Готово!\033[0m")
		fmt.Printf("\033[1;33m     %s\033[0m\n", shortUrl)
	}
}

// Функция для отображения статистики
func showStatistics() {
	store.mu.RLock()
	defer store.mu.RUnlock()

	if len(store.urls) == 0 {
		fmt.Println("\033[31mСтатистика пуста. Сначала создайте ссылки.\033[0m")
		return
	}

	fmt.Println("\n\033[1;35m--- ТЕКУЩАЯ СТАТИСТИКА --- \033[0m")
	fmt.Printf("%-8s | %-10s | %s\n", "Код", "Клики", "Оригинальный URL")
	fmt.Println("------------------------------------------------------")
	for code, url := range store.urls {
		fmt.Printf("%-8s | %-10d | %s\n", code, store.clicks[code], url)
	}
}

func saveUrl(originalURL string) string {
	code := generateShortCode()
	store.mu.Lock()
	store.urls[code] = originalURL
	store.clicks[code] = 0 // Устанавливаем начальное значение кликов
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

	store.mu.Lock() // Используем Lock, так как будем обновлять счетчик
	originalURL, ok := store.urls[code]
	if ok {
		store.clicks[code]++ // Увеличиваем счетчик при каждом переходе
	}
	store.mu.Unlock()

	if !ok {
		fmt.Printf("\n\033[31m[LOG %s] Ошибка: код %s не найден\033[0m\n", time.Now().Format("15:04:05"), code)
		http.NotFound(w, r)
		return
	}

	fmt.Printf("\n\033[34m[LOG %s] Переход по коду %s -> %s (Кликов: %d)\033[0m\n",
		time.Now().Format("15:04:05"), code, originalURL, store.clicks[code])

	http.Redirect(w, r, originalURL, http.StatusFound)
}
