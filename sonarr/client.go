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
	Monitor                  string `json:"monitor"` // "all", "future", etc.
}

type Series struct {
	Title             string     `json:"title"`
	TvdbID            int        `json:"tvdbId"`
	TitleSlug         string     `json:"titleSlug"`
	QualityProfileID  int        `json:"qualityProfileId"`
	RootFolderPath    string     `json:"rootFolderPath"`
	Monitored         bool       `json:"monitored"`
	SeasonFolder      bool       `json:"seasonFolder"`
	AddOptions        AddOptions `json:"addOptions"`
	SeriesType        string     `json:"seriesType"` // "standard", "daily", etc.
	Images            []Image    `json:"images,omitempty"`
	Tags              []int      `json:"tags,omitempty"`
	Year              int        `json:"year,omitempty"`
	LanguageProfileID int        `json:"languageProfileId,omitempty"` // optional
}

type Image struct {
	CoverType string `json:"coverType"`
	URL       string `json:"url"`
}

// AddSeriesByTMDB looks up series by TMDB ID and adds it to Sonarr
func (c *Client) AddSeriesByTMDB(tmdbID int, qualityProfileID int, rootFolder string) error {
	// Step 1: Lookup series by TMDB ID
	lookupURL := fmt.Sprintf("%s/api/v3/series/lookup?term=tmdb:%d", c.BaseURL, tmdbID)
	lookupReq, err := http.NewRequest("GET", lookupURL, nil)
	if err != nil {
		return err
	}
	lookupReq.Header.Set("X-Api-Key", c.APIKey)

	resp, err := http.DefaultClient.Do(lookupReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed lookup: %d - %s", resp.StatusCode, string(body))
	}

	var results []Series
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return err
	}
	if len(results) == 0 {
		return fmt.Errorf("no series found for tmdb id %d", tmdbID)
	}

	// Use the first result
	series := results[0]
	series.QualityProfileID = qualityProfileID
	series.RootFolderPath = rootFolder
	series.Monitored = true
	series.SeasonFolder = true
	series.SeriesType = "standard"
	series.AddOptions = AddOptions{
		SearchForMissingEpisodes: true,
		Monitor:                  "all",
	}

	// Step 2: POST to /series to add it
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

	postResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer postResp.Body.Close()

	if postResp.StatusCode != http.StatusCreated && postResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(postResp.Body)
		return fmt.Errorf("sonarr API returned status %d: %s", postResp.StatusCode, string(bodyBytes))
	}

	return nil
}
