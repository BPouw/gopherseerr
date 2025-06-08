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

// Client holds TMDB client config
type Client struct {
	APIKey string
}

// NewClient creates a new TMDB client
func NewClient(apiKey string) *Client {
	return &Client{APIKey: apiKey}
}

// Search searches TMDB for movies and tv shows by query
func (c *Client) Search(query string) ([]MediaBasic, error) {
	endpoint := fmt.Sprintf("%s/search/multi", baseURL)
	params := url.Values{}
	params.Set("api_key", c.APIKey)
	params.Set("query", query)
	params.Set("include_adult", "false")

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
