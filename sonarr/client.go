package sonarr

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
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
	Monitor                  string `json:"monitor"`
}

type SonarrSeason struct {
	SeasonNumber int  `json:"seasonNumber"`
	Monitored    bool `json:"monitored"`
}

type Series struct {
	ID                int            `json:"id,omitempty"`
	Title             string         `json:"title"`
	TvdbID            int            `json:"tvdbId"`
	TitleSlug         string         `json:"titleSlug"`
	QualityProfileID  int            `json:"qualityProfileId"`
	LanguageProfileID int            `json:"languageProfileId"`
	RootFolderPath    string         `json:"rootFolderPath"`
	Path              string         `json:"path,omitempty"`
	Monitored         bool           `json:"monitored"`
	SeasonFolder      bool           `json:"seasonFolder"`
	AddOptions        *AddOptions    `json:"addOptions,omitempty"`
	SeriesType        string         `json:"seriesType"`
	Images            []Image        `json:"images,omitempty"`
	Tags              []int          `json:"tags,omitempty"`
	Year              int            `json:"year,omitempty"`
	Seasons           []SonarrSeason `json:"seasons"`
}

type Image struct {
	CoverType string `json:"coverType"`
	URL       string `json:"url"`
}

type Episode struct {
	ID            int    `json:"id"`
	SeasonNumber  int    `json:"seasonNumber"`
	EpisodeNumber int    `json:"episodeNumber"`
	Title         string `json:"title"`
}

type CommandRequest struct {
	Name       string `json:"name"`
	EpisodeIDs []int  `json:"episodeIds"`
}

type AddSeriesOptions struct {
	TMDBID           int
	QualityProfileID int
	RootFolder       string
	SeasonsToMonitor map[int]bool
	AddEntireShow    bool
}

func (c *Client) AddSeries(opts AddSeriesOptions) (int, error) {
	lookupURL := fmt.Sprintf("%s/api/v3/series/lookup?term=tmdb:%d", c.BaseURL, opts.TMDBID)
	lookupReq, err := http.NewRequest("GET", lookupURL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create lookup request: %w", err)
	}
	lookupReq.Header.Set("X-Api-Key", c.APIKey)

	resp, err := http.DefaultClient.Do(lookupReq)
	if err != nil {
		return 0, fmt.Errorf("failed to execute lookup request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("failed series lookup, status %d: %s", resp.StatusCode, string(body))
	}

	var results []Series
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return 0, fmt.Errorf("failed to decode lookup response: %w", err)
	}
	if len(results) == 0 {
		return 0, fmt.Errorf("no series found for tmdb id %d", opts.TMDBID)
	}

	seriesToAdd := results[0]
	seriesToAdd.QualityProfileID = opts.QualityProfileID
	seriesToAdd.RootFolderPath = opts.RootFolder
	seriesToAdd.Monitored = true
	seriesToAdd.SeasonFolder = true
	seriesToAdd.LanguageProfileID = 1
	seriesToAdd.SeriesType = "standard"
	seriesToAdd.AddOptions = &AddOptions{
		SearchForMissingEpisodes: len(opts.SeasonsToMonitor) > 0 || opts.AddEntireShow,
		Monitor:                  "none",
	}

	for i := range seriesToAdd.Seasons {
		seasonNum := seriesToAdd.Seasons[i].SeasonNumber
		if seasonNum == 0 {
			seriesToAdd.Seasons[i].Monitored = false
			continue
		}
		if opts.AddEntireShow {
			seriesToAdd.Seasons[i].Monitored = true
		} else {
			_, shouldMonitor := opts.SeasonsToMonitor[seasonNum]
			seriesToAdd.Seasons[i].Monitored = shouldMonitor
		}
	}

	endpoint := fmt.Sprintf("%s/api/v3/series", c.BaseURL)
	payload, err := json.Marshal(seriesToAdd)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal series payload: %w", err)
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(payload))
	if err != nil {
		return 0, fmt.Errorf("failed to create add series request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", c.APIKey)

	postResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to execute add series request: %w", err)
	}
	defer postResp.Body.Close()

	bodyBytes, _ := io.ReadAll(postResp.Body)
	if postResp.StatusCode != http.StatusCreated {
		if strings.Contains(string(bodyBytes), "already been added") {
			return 0, errors.New("series already exists")
		}
		return 0, fmt.Errorf("sonarr API returned status %d: %s", postResp.StatusCode, string(bodyBytes))
	}

	var addedSeries Series
	if err := json.Unmarshal(bodyBytes, &addedSeries); err != nil {
		return 0, fmt.Errorf("failed to decode add series response: %w", err)
	}

	return addedSeries.ID, nil
}

func (c *Client) UpdateSeries(series *Series) error {
	series.AddOptions = nil

	endpoint := fmt.Sprintf("%s/api/v3/series/%d", c.BaseURL, series.ID)
	payload, err := json.Marshal(series)
	if err != nil {
		return fmt.Errorf("failed to marshal series payload for update: %w", err)
	}

	req, err := http.NewRequest("PUT", endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create update series request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", c.APIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute update series request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("sonarr update API returned status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (c *Client) GetSeriesByTMDB(tmdbID int) (*Series, error) {
	lookupURL := fmt.Sprintf("%s/api/v3/series/lookup?term=tmdb:%d", c.BaseURL, tmdbID)
	lookupReq, err := http.NewRequest("GET", lookupURL, nil)
	if err != nil {
		return nil, err
	}
	lookupReq.Header.Set("X-Api-Key", c.APIKey)
	lookupResp, err := http.DefaultClient.Do(lookupReq)
	if err != nil {
		return nil, err
	}
	defer lookupResp.Body.Close()

	if lookupResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDB ID %d not found via Sonarr lookup", tmdbID)
	}

	var lookupResults []Series
	if err := json.NewDecoder(lookupResp.Body).Decode(&lookupResults); err != nil || len(lookupResults) == 0 {
		return nil, fmt.Errorf("could not find series by TMDB ID %d in Sonarr", tmdbID)
	}
	targetTvdbID := lookupResults[0].TvdbID

	allSeriesURL := fmt.Sprintf("%s/api/v3/series", c.BaseURL)
	req, err := http.NewRequest("GET", allSeriesURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Api-Key", c.APIKey)

	allSeriesResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer allSeriesResp.Body.Close()

	var allSeries []Series
	if err := json.NewDecoder(allSeriesResp.Body).Decode(&allSeries); err != nil {
		return nil, err
	}

	for i, s := range allSeries {
		if s.TvdbID == targetTvdbID {
			return &allSeries[i], nil
		}
	}

	return nil, fmt.Errorf("series with TMDB ID %d not found in Sonarr library", tmdbID)
}

func (c *Client) SearchEpisodes(episodeIDs []int) error {
	endpoint := fmt.Sprintf("%s/api/v3/command", c.BaseURL)
	cmd := CommandRequest{
		Name:       "EpisodeSearch",
		EpisodeIDs: episodeIDs,
	}
	payload, err := json.Marshal(cmd)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", c.APIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("command failed with status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (c *Client) GetEpisodes(seriesID int) ([]Episode, error) {
	endpoint := fmt.Sprintf("%s/api/v3/episode?seriesId=%d", c.BaseURL, seriesID)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Api-Key", c.APIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var episodes []Episode
	if err := json.NewDecoder(resp.Body).Decode(&episodes); err != nil {
		return nil, err
	}
	return episodes, nil
}
