package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

//go:embed all:public
var embeddedFS embed.FS

func main() {
	var staticFS fs.FS
	if dir := os.Getenv("STATIC_DIR"); dir != "" {
		// Dev: serve from disk — no rebuild needed for frontend changes
		log.Printf("Serving static files from disk: %s", dir)
		staticFS = os.DirFS(dir)
	} else {
		// Prod: serve from embedded FS (files baked into binary)
		log.Println("Serving static files from embedded FS")
		sub, err := fs.Sub(embeddedFS, "public")
		if err != nil {
			log.Fatalf("Failed to access embedded public dir: %v", err)
		}
		staticFS = sub
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL != "" {
		log.Printf("DB_URL is set: %s", dbURL)
	} else {
		log.Println("DB_URL is not set — database not connected")
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	fileServer := http.FileServerFS(staticFS)
	r.Handle("/*", fileServer)

	log.Printf("Yumyums HQ server listening on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
