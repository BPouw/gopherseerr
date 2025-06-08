package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/bpouw/gopherseerr/radarr"
	"github.com/bpouw/gopherseerr/sonarr"
	"github.com/bpouw/gopherseerr/tmdb"
)

var (
	config       Config
	templates    *template.Template
	tmdbClient   *tmdb.Client
	radarrClient *radarr.Client
	sonarrClient *sonarr.Client
)

type Config struct {
	Port         string `json:"port"`
	TMDBApiKey   string `json:"tmdb_api_key"`
	RadarrURL    string `json:"radarr_url"`
	RadarrApiKey string `json:"radarr_api_key"`
	SonarrURL    string `json:"sonarr_url"`
	SonarrApiKey string `json:"sonarr_api_key"`
}

func main() {
	// Load config
	f, err := os.Open("config.json")
	if err != nil {
		log.Fatal("Error loading config:", err)
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&config); err != nil {
		log.Fatal("Error parsing config:", err)
	}

	templates = template.Must(template.ParseGlob("templates/*.gohtml"))

	tmdbClient = tmdb.NewClient(config.TMDBApiKey)

	radarrClient = radarr.NewClient(config.RadarrURL, config.RadarrApiKey)
	sonarrClient = sonarr.NewClient(config.SonarrURL, config.SonarrApiKey)

	http.HandleFunc("/", handleSearch)
	http.HandleFunc("/request", handleRequest)

	log.Println("Starting server on port", config.Port)
	log.Fatal(http.ListenAndServe(":"+config.Port, nil))
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		templates.ExecuteTemplate(w, "search.gohtml", nil)
		return
	}

	results, err := tmdbClient.Search(q)
	if err != nil {
		http.Error(w, "TMDB search error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	templates.ExecuteTemplate(w, "results.gohtml", results)
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	mediaType := r.FormValue("type") // movie or tv
	tmdbIDStr := r.FormValue("tmdb_id")

	tmdbID, err := strconv.Atoi(tmdbIDStr)
	if err != nil {
		http.Error(w, "Invalid tmdb_id", http.StatusBadRequest)
		return
	}

	// example root folders, adjust to your setup
	var rootFolder = ""
	var errAdd error

	switch mediaType {
	case "movie":
		rootFolder = "X:\\plex\\movies" // your movies folder path
		errAdd = radarrClient.AddMovieByTMDB(tmdbID, 7, rootFolder)
	case "tv":
		rootFolder = "X:\\plex\\shows" // your shows folder path
		errAdd = sonarrClient.AddSeriesByTMDB(tmdbID, 7, rootFolder)
	default:
		http.Error(w, "Unsupported media type", http.StatusBadRequest)
		return
	}

	if errAdd != nil {
		http.Error(w, "Failed to add request: "+errAdd.Error(), http.StatusInternalServerError)
		return
	}

	showPopupAndRedirect(w, "Request successfully submitted!", "/")
}

func showPopupAndRedirect(w http.ResponseWriter, message, redirectURL string) {
	w.Header().Set("Content-Type", "text/html")
	html := fmt.Sprintf(`
        <!DOCTYPE html>
        <html lang="en">
        <head><meta charset="UTF-8"><title>Notification</title></head>
        <body>
        <script>
            alert(%q);
            window.location.href = %q;
        </script>
        </body>
        </html>`, message, redirectURL)
	w.Write([]byte(html))
}
