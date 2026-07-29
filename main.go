package main

import (
	"code3-inventory/internal/api"
	"code3-inventory/internal/db"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

//go:embed static
var staticFS embed.FS

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		tmpDir, err := os.MkdirTemp("", "inventory")
		if err != nil {
			log.Fatal(err)
		}
		dbPath = filepath.Join(tmpDir, "inventory.db")
	}

	store, err := db.New(dbPath)
	if err != nil {
		log.Fatal("数据库初始化失败:", err)
	}
	defer store.Close()

	if err := store.Init(); err != nil {
		log.Fatal("数据库建表失败:", err)
	}

	a := api.New(store)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/skus", a.ListSKUs)
	mux.HandleFunc("POST /api/skus", a.CreateSKU)
	mux.HandleFunc("GET /api/skus/{id}", a.GetSKU)
	mux.HandleFunc("PUT /api/skus/{id}", a.UpdateSKU)
	mux.HandleFunc("DELETE /api/skus/{id}", a.DeleteSKU)
	mux.HandleFunc("POST /api/inbound", a.Inbound)
	mux.HandleFunc("POST /api/outbound", a.Outbound)
	mux.HandleFunc("GET /api/inventory", a.Inventory)
	mux.HandleFunc("GET /api/alerts", a.Alerts)
	mux.HandleFunc("GET /api/ledger", a.Ledger)
	mux.HandleFunc("GET /api/stats", a.Stats)

	// SPA fallback
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			serveSPA(w, r)
		}
	})

	// CORS
	handler := corsMiddleware(mux)

	addr := ":" + port
	log.Printf("🚀 服务启动 http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}

func serveSPA(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/" || path == "" {
		path = "/index.html"
	}
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	f, err := sub.Open(strings.TrimPrefix(path, "/"))
	if err != nil {
		// SPA fallback: serve index.html
		idx, err2 := sub.Open("index.html")
		if err2 != nil {
			http.NotFound(w, r)
			return
		}
		defer idx.Close()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeContent(w, r, "index.html", idx.ModTime(), idx)
		return
	}
	defer f.Close()
	info, _ := f.Stat()
	if info.IsDir() {
		http.NotFound(w, r)
		return
	}
	ctype := contentType(path)
	w.Header().Set("Content-Type", ctype)
	http.ServeContent(w, r, path, info.ModTime(), f)
}

func contentType(path string) string {
	switch {
	case strings.HasSuffix(path, ".html"):
		return "text/html; charset=utf-8"
	case strings.HasSuffix(path, ".css"):
		return "text/css; charset=utf-8"
	case strings.HasSuffix(path, ".js"):
		return "application/javascript; charset=utf-8"
	default:
		return "application/octet-stream"
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func readJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

func param(r *http.Request, key string) string {
	return r.PathValue(key)
}

func mustInt64(s string) int64 {
	var n int64
	fmt.Sscanf(s, "%d", &n)
	return n
}

func mustFloat64(s string) float64 {
	var n float64
	fmt.Sscanf(s, "%f", &n)
	return n
}