package sonarr

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
	SearchForMissingEpisodes bool   `json:"searchForMissingEpisodes"`
	Monitor                  string `json:"monitor"` // e.g., "all"
}

type Series struct {
	Title        string     `json:"title,omitempty"`
	TvdbID       int        `json:"tvdbId,omitempty"`
	TmdbID       int        `json:"tmdbId"`
	Quality      int        `json:"qualityProfileId,omitempty"`
	RootFolder   string     `json:"rootFolderPath,omitempty"`
	Monitored    bool       `json:"monitored"`
	SeasonFolder bool       `json:"seasonFolder"`
	AddOptions   AddOptions `json:"addOptions"`
	SeriesType   string     `json:"seriesType,omitempty"` // e.g., "standard"
}

func (c *Client) AddSeriesByTMDB(tmdbID int, qualityProfileID int, rootFolder string) error {
	series := Series{
		TmdbID:       tmdbID,
		Quality:      qualityProfileID,
		RootFolder:   rootFolder,
		Monitored:    true,
		SeasonFolder: true,
		AddOptions: AddOptions{
			SearchForMissingEpisodes: true,
			Monitor:                  "all",
		},
		SeriesType: "standard", // or "anime", "daily" if needed
	}

	endpoint := fmt.Sprintf("%s/api/v3/series", c.BaseURL)
	u, err := url.Parse(endpoint)
	if err != nil {
		return err
	}
	query := u.Query()
	query.Set("apikey", c.APIKey)
	u.RawQuery = query.Encode()

	payload, err := json.Marshal(series)
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
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("sonarr API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}
