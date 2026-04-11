package main

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"os"
)

//go:embed templates/*
var templateFS embed.FS

func main() {
	apiURL := os.Getenv("API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:8090"
	}

	// Derive the WebSocket URL from the HTTP URL.
	wsURL := "ws" + apiURL[4:] // http:// → ws://, https:// → wss://

	tmpl, err := template.ParseFS(templateFS, "templates/*.html")
	if err != nil {
		log.Fatalf("Failed to parse templates: %v", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		data := map[string]string{
			"APIURL": apiURL,
			"WSURL":  wsURL,
		}
		if err := tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
			log.Printf("Template error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("GET /calories", func(w http.ResponseWriter, r *http.Request) {
		data := map[string]string{
			"APIURL": apiURL,
			"WSURL":  wsURL,
		}
		if err := tmpl.ExecuteTemplate(w, "calories.html", data); err != nil {
			log.Printf("Template error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	})

	log.Println("Web server starting on :3080")
	if err := http.ListenAndServe(":3080", mux); err != nil {
		log.Fatal(err)
	}
}
