package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"shortenergo/internal/api"
	"shortenergo/internal/console"
	"shortenergo/internal/storage"
)

func main() {
	store := storage.NewStore()

	h := api.NewHandler(store)

	mux := http.NewServeMux()

	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/assets/", fs)

	mux.HandleFunc("/shorten", h.ShortenHandler)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, "./static/index.html")
			return
		}
		h.RedirectHandler(w, r)
	})

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go console.RunConsole(store)

	go func() {
		log.Println("Сервер: http://localhost:8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Ошибка: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}
