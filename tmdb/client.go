package tmdb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

const baseURL = "https://api.themoviedb.org/3"

type SearchResult struct {
	Page         int          `json:"page"`
	Results      []MediaBasic `json:"results"`
	TotalResults int          `json:"total_results"`
	TotalPages   int          `json:"total_pages"`
}

type MediaBasic struct {
	ID           int    `json:"id"`
	Title        string `json:"title,omitempty"`      // movie title
	Name         string `json:"name,omitempty"`       // tv show name
	MediaType    string `json:"media_type,omitempty"` // "movie" or "tv"
	Overview     string `json:"overview"`
	PosterPath   string `json:"poster_path"`
	ReleaseDate  string `json:"release_date,omitempty"`
	FirstAirDate string `json:"first_air_date,omitempty"`
}

type TVShowDetails struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Overview     string   `json:"overview"`
	PosterPath   string   `json:"poster_path"`
	FirstAirDate string   `json:"first_air_date"`
	Seasons      []Season `json:"seasons"`
}

type Season struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	SeasonNumber int    `json:"season_number"`
	EpisodeCount int    `json:"episode_count"`
	AirDate      string `json:"air_date"`
	PosterPath   string `json:"poster_path"`
}

type SeasonDetails struct {
	Episodes []TMDbEpisode `json:"episodes"`
}

type TMDbEpisode struct {
	EpisodeNumber int    `json:"episode_number"`
	Name          string `json:"name"`
	Overview      string `json:"overview"`
}

type Client struct {
	APIKey string
}

func NewClient(apiKey string) *Client {
	return &Client{APIKey: apiKey}
}

func (c *Client) Search(query string) ([]MediaBasic, error) {
	endpoint := fmt.Sprintf("%s/search/multi", baseURL)
	params := url.Values{"api_key": {c.APIKey}, "query": {query}, "include_adult": {"false"}}
	fullURL := fmt.Sprintf("%s?%s", endpoint, params.Encode())

	resp, err := http.Get(fullURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	filtered := []MediaBasic{}
	for _, item := range result.Results {
		if item.MediaType == "movie" || item.MediaType == "tv" {
			filtered = append(filtered, item)
		}
	}
	return filtered, nil
}

func (c *Client) GetTVShowDetails(tvID int) (*TVShowDetails, error) {
	endpoint := fmt.Sprintf("%s/tv/%d", baseURL, tvID)
	params := url.Values{"api_key": {c.APIKey}}
	fullURL := fmt.Sprintf("%s?%s", endpoint, params.Encode())

	resp, err := http.Get(fullURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDB API returned non-200 status: %d", resp.StatusCode)
	}

	var details TVShowDetails
	if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
		return nil, err
	}
	return &details, nil
}

// GetSeasonDetails fetches episode information for a specific season.
func (c *Client) GetSeasonDetails(tvID int, seasonNumber int) (*SeasonDetails, error) {
	endpoint := fmt.Sprintf("%s/tv/%d/season/%d", baseURL, tvID, seasonNumber)
	params := url.Values{"api_key": {c.APIKey}}
	fullURL := fmt.Sprintf("%s?%s", endpoint, params.Encode())

	resp, err := http.Get(fullURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDB API returned non-200 status for season details: %d", resp.StatusCode)
	}

	var details SeasonDetails
	if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
		return nil, err
	}
	return &details, nil
}
