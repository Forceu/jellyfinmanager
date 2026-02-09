package tvdb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/forceu/jellyfinmanager/models"
)

const (
	baseURL = "https://api4.thetvdb.com/v4"
)

// Client handles API interactions with TVDB
type Client struct {
	apiKey     string
	token      string
	httpClient *http.Client
}

// NewClient creates a new TVDB API client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Login authenticates with TVDB and gets a bearer token
func (c *Client) Login() error {
	payload := map[string]string{
		"apikey": c.apiKey,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling login payload: %w", err)
	}

	req, err := http.NewRequest("POST", baseURL+"/login", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating login request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing login request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
		Status string `json:"status"`
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return fmt.Errorf("decoding login response: %w", err)
	}

	c.token = result.Data.Token
	return nil
}

// makeRequest performs an authenticated request to TVDB API
func (c *Client) makeRequest(method, endpoint string) (*http.Response, error) {
	if c.token == "" {
		return nil, fmt.Errorf("not authenticated - call Login() first")
	}

	req, err := http.NewRequest(method, baseURL+endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}

	return resp, nil
}

// SearchSeriesByTVDBID searches for a series by TVDB ID
func (c *Client) SearchSeriesByTVDBID(tvdbID string) (*SeriesExtended, error) {
	resp, err := c.makeRequest("GET", "/series/"+tvdbID+"/extended")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("series not found (status %d)", resp.StatusCode)
	}

	var result struct {
		Data   SeriesExtended `json:"data"`
		Status string         `json:"status"`
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("decoding series response: %w", err)
	}

	return &result.Data, nil
}

// GetSeriesEpisodes retrieves all episodes for a series
func (c *Client) GetSeriesEpisodes(seriesID string) ([]Episode, error) {
	var allEpisodes []Episode
	page := 0

	for {
		endpoint := fmt.Sprintf("/series/%s/episodes/default?page=%d", seriesID, page)
		resp, err := c.makeRequest("GET", endpoint)
		if err != nil {
			return nil, err
		}

		var result struct {
			Data struct {
				Episodes []Episode `json:"episodes"`
			} `json:"data"`
			Links struct {
				Next string `json:"next"`
			} `json:"links"`
			Status string `json:"status"`
		}

		err = json.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("decoding episodes response: %w", err)
		}
		resp.Body.Close()

		allEpisodes = append(allEpisodes, result.Data.Episodes...)

		// Check if there are more pages
		if result.Links.Next == "" {
			break
		}
		page++
	}

	return allEpisodes, nil
}

// SeriesExtended represents extended series information from TVDB
type SeriesExtended struct {
	ID                int      `json:"id"`
	Name              string   `json:"name"`
	Slug              string   `json:"slug"`
	Overview          string   `json:"overview"`
	Year              string   `json:"year"`
	FirstAired        string   `json:"firstAired"`
	LastAired         string   `json:"lastAired"`
	Status            Status   `json:"status"`
	DefaultSeasonType int      `json:"defaultSeasonType"`
	Seasons           []Season `json:"seasons"`
}

// Season represents a season from TVDB
type Season struct {
	ID     int    `json:"id"`
	Number int    `json:"number"`
	Name   string `json:"name"`
	Type   struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"type"`
}

// Status represents a series status
type Status struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Episode represents an episode from TVDB
type Episode struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	Overview       string `json:"overview"`
	Aired          string `json:"aired"`
	SeasonNumber   int    `json:"seasonNumber"`
	Number         int    `json:"number"`
	RuntimeMinutes int    `json:"runtime"`
	SeriesID       int    `json:"seriesId"`
	AbsoluteNumber int    `json:"absoluteNumber"`
	FinaleType     string `json:"finaleType"`
}

// FindMissingEpisodes finds episodes that are missing from Jellyfin
// It also excludes multi-part episodes that appear merged based on runtime analysis
func FindMissingEpisodes(tvdbEpisodes []Episode, jellyfinEpisodes map[string]int, checkSpecials bool) []models.MissingEpisode {
	var missing []models.MissingEpisode

	// State variables to track merging of multi-part episodes
	var (
		chainActive           bool
		chainSeason           int
		chainJfRuntime        int // Actual runtime of the file in Jellyfin
		chainTvdbRuntimeAccum int // Expected runtime (sum of TVDB episodes in this chain)
	)

	for _, ep := range tvdbEpisodes {
		// Create key for comparison (season:episode)
		key := fmt.Sprintf("%d:%d", ep.SeasonNumber, ep.Number)

		airDate, errDate := time.Parse("2006-01-02", ep.Aired)
		jfRuntime, episodeStored := jellyfinEpisodes[key]

		if episodeStored {
			// Episode found in Jellyfin.
			// Start a new chain: this file might contain subsequent missing episodes.
			chainActive = true
			chainSeason = ep.SeasonNumber
			chainJfRuntime = jfRuntime
			chainTvdbRuntimeAccum = ep.RuntimeMinutes
		} else {
			// Episode NOT found in Jellyfin.
			// Check if it is a valid candidate for being reported as missing.
			if (ep.SeasonNumber != 0 || checkSpecials) &&
				(errDate == nil && airDate.Before(time.Now())) {

				isMerged := false

				// CHECK MERGE CONDITION:
				// 1. We must have a valid previous episode (chainActive)
				// 2. It must be in the same season
				if chainActive && chainSeason == ep.SeasonNumber {
					// Add current episode's expected length to the accumulator
					chainTvdbRuntimeAccum += ep.RuntimeMinutes

					// Check if the file on disk is at least 85% of the total expected length
					requiredLength := float64(chainTvdbRuntimeAccum)
					if float64(chainJfRuntime) >= requiredLength*0.85 {
						isMerged = true
					} else {
						// The file is not long enough to include this episode.
						// The chain is broken.
						chainActive = false
					}
				} else {
					// No active chain or season mismatch
					chainActive = false
				}

				// If not merged, mark as missing
				if !isMerged {
					missing = append(missing, models.MissingEpisode{
						SeasonNumber:  ep.SeasonNumber,
						EpisodeNumber: ep.Number,
						EpisodeName:   ep.Name,
						AirDate:       ep.Aired,
						Overview:      ep.Overview,
					})
					// A missing episode breaks the chain for subsequent episodes
					chainActive = false
				}
			}
		}
	}

	return missing
}
