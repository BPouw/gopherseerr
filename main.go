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
	Port             string `json:"port"`
	TMDBApiKey       string `json:"tmdb_api_key"`
	RadarrURL        string `json:"radarr_url"`
	RadarrApiKey     string `json:"radarr_api_key"`
	SonarrURL        string `json:"sonarr_url"`
	SonarrApiKey     string `json:"sonarr_api_key"`
	RadarrRootFolder string `json:"radarr_root_folder"`
	SonarrRootFolder string `json:"sonarr_root_folder"`
}

func main() {
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
	http.HandleFunc("/show", handleShowDetails)
	http.HandleFunc("/episodes", handleGetEpisodes)
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

func handleShowDetails(w http.ResponseWriter, r *http.Request) {
	tmdbIDStr := r.URL.Query().Get("tmdb_id")
	tmdbID, err := strconv.Atoi(tmdbIDStr)
	if err != nil {
		http.Error(w, "Invalid tmdb_id", http.StatusBadRequest)
		return
	}
	showDetails, err := tmdbClient.GetTVShowDetails(tmdbID)
	if err != nil {
		http.Error(w, "Failed to get show details from TMDB: "+err.Error(), http.StatusInternalServerError)
		return
	}
	err = templates.ExecuteTemplate(w, "show.gohtml", showDetails)
	if err != nil {
		http.Error(w, "Failed to render template: "+err.Error(), http.StatusInternalServerError)
	}
}

func handleGetEpisodes(w http.ResponseWriter, r *http.Request) {
	tmdbIDStr := r.URL.Query().Get("tmdb_id")
	seasonNumberStr := r.URL.Query().Get("season")
	tmdbID, err1 := strconv.Atoi(tmdbIDStr)
	seasonNumber, err2 := strconv.Atoi(seasonNumberStr)
	if err1 != nil || err2 != nil {
		http.Error(w, "Invalid tmdb_id or season number", http.StatusBadRequest)
		return
	}

	seasonDetails, err := tmdbClient.GetSeasonDetails(tmdbID, seasonNumber)
	if err != nil {
		http.Error(w, "Failed to get season details: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(seasonDetails.Episodes)
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	r.ParseForm()

	mediaType := r.FormValue("type")
	tmdbIDStr := r.FormValue("tmdb_id")
	tmdbID, err := strconv.Atoi(tmdbIDStr)
	if err != nil {
		http.Error(w, "Invalid tmdb_id", http.StatusBadRequest)
		return
	}

	var errAdd error
	var successMessage string

	switch mediaType {
	case "movie":
		rootFolder := config.RadarrRootFolder
		errAdd = radarrClient.AddMovieByTMDB(tmdbID, 7, rootFolder)
		successMessage = "Movie request successfully submitted!"

	case "tv":
		rootFolder := config.SonarrRootFolder
		requestType := r.FormValue("request_type")

		opts := sonarr.AddSeriesOptions{
			TMDBID:           tmdbID,
			QualityProfileID: 7,
			RootFolder:       rootFolder,
			SeasonsToMonitor: make(map[int]bool),
		}

		switch requestType {
		case "full_show":
			opts.AddEntireShow = true
			_, errAdd = sonarrClient.AddSeries(opts)
			successMessage = "Request to add the full show has been submitted!"
			if errAdd != nil && errAdd.Error() == "series already exists" {
				// Handle existing series: ensure all seasons are monitored
				log.Println("Series exists, ensuring all seasons are monitored...")
				series, findErr := sonarrClient.GetSeriesByTMDB(tmdbID)
				if findErr != nil {
					http.Error(w, "Series exists, but failed to look it up: "+findErr.Error(), http.StatusInternalServerError)
					return
				}
				for i := range series.Seasons {
					if series.Seasons[i].SeasonNumber > 0 {
						series.Seasons[i].Monitored = true
					}
				}
				errAdd = sonarrClient.UpdateSeries(series)
			}

		case "season":
			seasonNumberStr := r.FormValue("season_number")
			seasonNumber, err := strconv.Atoi(seasonNumberStr)
			if err != nil {
				http.Error(w, "Invalid season_number", http.StatusBadRequest)
				return
			}
			opts.SeasonsToMonitor[seasonNumber] = true
			_, errAdd = sonarrClient.AddSeries(opts)
			successMessage = fmt.Sprintf("Request to add Season %d has been submitted!", seasonNumber)
			if errAdd != nil && errAdd.Error() == "series already exists" {
				log.Printf("Series exists, ensuring season %d is monitored...", seasonNumber)
				series, findErr := sonarrClient.GetSeriesByTMDB(tmdbID)
				if findErr != nil {
					http.Error(w, "Series exists, but failed to look it up: "+findErr.Error(), http.StatusInternalServerError)
					return
				}
				var seasonUpdated = false
				for i := range series.Seasons {
					if series.Seasons[i].SeasonNumber == seasonNumber {
						series.Seasons[i].Monitored = true
						seasonUpdated = true
						break
					}
				}
				if seasonUpdated {
					errAdd = sonarrClient.UpdateSeries(series)
				} else {
					errAdd = fmt.Errorf("could not find season %d in existing series to update", seasonNumber)
				}
			}

		case "episode":
			seasonNumberStr := r.FormValue("season_number")
			episodeNumberStr := r.FormValue("episode_number")
			seasonNumber, err1 := strconv.Atoi(seasonNumberStr)
			episodeNumber, err2 := strconv.Atoi(episodeNumberStr)
			if err1 != nil || err2 != nil {
				http.Error(w, "Invalid season or episode number", http.StatusBadRequest)
				return
			}

			// Try to find the series first.
			series, err := sonarrClient.GetSeriesByTMDB(tmdbID)
			if err != nil {
				// If not found, add it for the first time.
				log.Println("Series not found in Sonarr, adding it...")
				opts.SeasonsToMonitor[seasonNumber] = true // Monitor the season
				id, addErr := sonarrClient.AddSeries(opts)
				if addErr != nil {
					http.Error(w, "Failed to add new series for episode request: "+addErr.Error(), http.StatusInternalServerError)
					return
				}
				// After adding, we need to fetch it again to get the full series object
				series, err = sonarrClient.GetSeriesByTMDB(tmdbID)
				if err != nil {
					http.Error(w, "Added series but could not immediately re-fetch it: "+err.Error(), http.StatusInternalServerError)
					return
				}
				series.ID = id
			} else {
				// Series exists, ensure the season is monitored.
				log.Println("Series found in Sonarr, ensuring season is monitored...")
				var needsUpdate = false
				for i := range series.Seasons {
					if series.Seasons[i].SeasonNumber == seasonNumber && !series.Seasons[i].Monitored {
						series.Seasons[i].Monitored = true
						needsUpdate = true
						break
					}
				}
				if needsUpdate {
					log.Println("Updating series to monitor new season...")
					if updateErr := sonarrClient.UpdateSeries(series); updateErr != nil {
						http.Error(w, "Failed to update series monitoring status: "+updateErr.Error(), http.StatusInternalServerError)
						return
					}
				}
			}

			// Now that the series exists and the season is monitored, search for the episode.
			allEpisodes, epErr := sonarrClient.GetEpisodes(series.ID)
			if epErr != nil {
				http.Error(w, "Failed to get episodes from Sonarr: "+epErr.Error(), http.StatusInternalServerError)
				return
			}
			var targetEpisodeID = -1
			for _, ep := range allEpisodes {
				if ep.SeasonNumber == seasonNumber && ep.EpisodeNumber == episodeNumber {
					targetEpisodeID = ep.ID
					break
				}
			}

			if targetEpisodeID == -1 {
				http.Error(w, "Could not find the specified episode in Sonarr.", http.StatusInternalServerError)
				return
			}

			errAdd = sonarrClient.SearchEpisodes([]int{targetEpisodeID})
			successMessage = fmt.Sprintf("Search for S%02dE%02d has been triggered!", seasonNumber, episodeNumber)

		default:
			http.Error(w, "Unsupported TV request type", http.StatusBadRequest)
			return
		}
	default:
		http.Error(w, "Unsupported media type", http.StatusBadRequest)
		return
	}

	if errAdd != nil {
		http.Error(w, "Failed to process request: "+errAdd.Error(), http.StatusInternalServerError)
		return
	}
	redirectURL := r.Header.Get("Referer")
	if redirectURL == "" {
		redirectURL = "/"
	}
	showPopupAndRedirect(w, successMessage, redirectURL)
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
