// im learning go dont judge the code

package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Snippet struct {
	ID        string
	Title     string
	Content   string
	Language  string
	CreatedAt time.Time
	ExpiresAt time.Time
}

type PageData struct {
	Snippet   *Snippet
	Remaining string
}

type App struct {
	mu       sync.RWMutex
	snippets map[string]Snippet
}

func generateID(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)[:length]
}

func renderTemplate(w http.ResponseWriter, tmplName string, data PageData) {
	files := []string{
		filepath.Join("templates", "layout.html"),
		filepath.Join("templates", tmplName),
	}

	tmpl, err := template.ParseFiles(files...)
	if err != nil {
		http.Error(w, "Template Parsing Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, "Template Execution Error: "+err.Error(), http.StatusInternalServerError)
	}
}

func (app *App) handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	renderTemplate(w, "create.html", PageData{})
}

func (app *App) handleSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	content := r.FormValue("content")
	language := r.FormValue("language")
	ttlStr := r.FormValue("ttl")

	if title == "" || content == "" {
		http.Error(w, "Title and content cannot be empty", http.StatusBadRequest)
		return
	}

	ttl, err := time.ParseDuration(ttlStr)
	if err != nil {
		ttl = 24 * time.Hour
	}

	id := generateID(6)
	now := time.Now()

	snippet := Snippet{
		ID:        id,
		Title:     title,
		Content:   content,
		Language:  language,
		CreatedAt: now,
		ExpiresAt: now.Add(ttl),
	}

	app.mu.Lock()
	app.snippets[id] = snippet
	app.mu.Unlock()

	http.Redirect(w, r, "/v/"+id, http.StatusSeeOther)
}

func (app *App) handleView(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/v/")

	app.mu.RLock()
	snippet, exists := app.snippets[id]
	app.mu.RUnlock()

	if !exists || time.Now().After(snippet.ExpiresAt) {
		http.Error(w, "Snippet not found or has expired.", http.StatusNotFound)
		return
	}

	data := PageData{
		Snippet:   &snippet,
		Remaining: time.Until(snippet.ExpiresAt).Round(time.Minute).String(),
	}

	renderTemplate(w, "view.html", data)
}

func (app *App) handleRaw(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/raw/")

	app.mu.RLock()
	snippet, exists := app.snippets[id]
	app.mu.RUnlock()

	if !exists || time.Now().After(snippet.ExpiresAt) {
		http.Error(w, "Snippet not found or expired", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(snippet.Content))
}

func (app *App) startJanitor() {
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for range ticker.C {
			app.mu.Lock()
			now := time.Now()
			for id, snip := range app.snippets {
				if now.After(snip.ExpiresAt) {
					delete(app.snippets, id)
				}
			}
			app.mu.Unlock()
		}
	}()
}

func main() {
	app := &App{
		snippets: make(map[string]Snippet),
	}

	app.startJanitor()

	mux := http.NewServeMux()

	// Serve static files (CSS & JS)
	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Application routes
	mux.HandleFunc("/", app.handleHome)
	mux.HandleFunc("/save", app.handleSave)
	mux.HandleFunc("/v/", app.handleView)
	mux.HandleFunc("/raw/", app.handleRaw)

	port := ":8080"
	fmt.Printf("⚡ SnipBin running at http://localhost%s\n", port)
	log.Fatal(http.ListenAndServe(port, mux))
}
