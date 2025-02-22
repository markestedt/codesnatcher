package main

import (
	"embed"
	"errors"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/markestedt/codesnatcher/internal/ai"
)

//go:embed static
var static embed.FS

//go:embed templates
var templates embed.FS

type application struct {
	templates *template.Template
	aiService *ai.Service
}

func main() {
	LoadEnv()

	t := template.Must(template.ParseFS(templates, "templates/*.html"))

	app := &application{
		templates: t,
		aiService: &ai.Service{},
	}

	fSys, err := fs.Sub(static, ".")
	if err != nil {
		log.Printf("Failed to load static files: %s", err)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.FileServer(http.FS(fSys)))

	mux.HandleFunc("GET /", app.getIndexHandler)
	mux.HandleFunc("POST /image", app.postImageHandler)

	err = http.ListenAndServe(":9292", mux)
	if errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server closed\n")
	} else if err != nil {
		log.Fatalf("error starting server: %s\n", err)
	}
}

func LoadEnv() {
	p, err := os.Executable()

	if err != nil {
		log.Fatal(err)
	}

	p = filepath.Dir(p)
	err = godotenv.Load(path.Join(p, ".env"))

	if err != nil {
		log.Printf("Error loading .env file from %s", p)
	} else {
		return
	}

	err = godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}
