package console

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"shortenergo/internal/storage"
	"shortenergo/internal/utils"
)

func RunConsole(store *storage.Store) {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("Консоль запущена. Введите URL или команду (stat, top).")

	for {
		fmt.Print("\n> ")
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
				showUrlStats(store, parts[1])
			} else {
				showStatistics(store)
			}
			continue
		case "top":
			if len(parts) == 2 {
				showTop(store, parts[1])
			}
			continue
		}

		cleanURL, err := utils.ValidateURL(input)
		if err != nil {
			fmt.Printf("Ошибка: %v\n", err)
			continue
		}

		code := store.Save(cleanURL)
		fmt.Printf("Готово: http://localhost:8080/%s\n", code)
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("02.01 15:04")
}

func showStatistics(store *storage.Store) {
	data := store.GetAll()
	if len(data) == 0 {
		fmt.Println("Пусто")
		return
	}
	fmt.Printf("%-8s | %-6s | %s\n", "Код", "Клики", "URL")
	for code, s := range data {
		fmt.Printf("%-8s | %-6d | %s\n", code, s.Clicks, s.URL)
	}
}

func showUrlStats(store *storage.Store, code string) {
	s, ok := store.Get(code)
	if !ok {
		fmt.Println("Код не найден")
		return
	}
	fmt.Printf("URL: %s (Клики: %d)\n", s.URL, s.Clicks)

}

func showTop(store *storage.Store, nStr string) {
	var n int
	fmt.Sscanf(nStr, "%d", &n)
	data := store.GetAll()

	type pair struct {
		Code   string
		Clicks int
	}
	list := make([]pair, 0, len(data))
	for code, s := range data {
		list = append(list, pair{code, s.Clicks})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Clicks > list[j].Clicks })
	if len(list) > n {
		list = list[:n]
	}
	for i, p := range list {
		fmt.Printf("%d. %s — %d\n", i+1, p.Code, p.Clicks)
	}
}
