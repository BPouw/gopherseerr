package radarr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Client struct {
	BaseURL string
	APIKey  string
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
	}
}

type AddOptions struct {
	SearchForMovie bool   `json:"searchForMovie"`
	Monitor        string `json:"monitor"` // typically "movieOnly"
}

type Movie struct {
	Title               string     `json:"title,omitempty"`
	TmdbID              int        `json:"tmdbId"`
	Quality             int        `json:"qualityProfileId,omitempty"`
	RootFolder          string     `json:"rootFolderPath,omitempty"`
	Monitored           bool       `json:"monitored"`
	AddOptions          AddOptions `json:"addOptions"`
	MinimumAvailability string     `json:"minimumAvailability,omitempty"` // Optional: e.g., "released"
}

func (c *Client) AddMovieByTMDB(tmdbID int, qualityProfileID int, rootFolder string) error {
	movie := Movie{
		TmdbID:     tmdbID,
		Quality:    qualityProfileID,
		RootFolder: rootFolder,
		Monitored:  true,
		AddOptions: AddOptions{
			SearchForMovie: true,
			Monitor:        "movieOnly",
		},
		MinimumAvailability: "released", // Optional, avoids grabbing pre-releases
	}

	endpoint := fmt.Sprintf("%s/api/v3/movie", c.BaseURL)
	u, err := url.Parse(endpoint)
	if err != nil {
		return err
	}
	query := u.Query()
	query.Set("apikey", c.APIKey)
	u.RawQuery = query.Encode()

	payload, err := json.Marshal(movie)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", u.String(), bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		bodyBytes, err2 := io.ReadAll(resp.Body)
		if err2 != nil {
			return fmt.Errorf("radarr API returned status %d and error reading response body: %v", resp.StatusCode, err2)
		}
		return fmt.Errorf("radarr API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}
