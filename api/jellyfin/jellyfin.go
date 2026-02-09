package jellyfin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/forceu/jellyfinmanager/models"
)

// Client handles API interactions with Jellyfin
type Client struct {
	config     models.Config
	httpClient *http.Client
}

// NewClient creates a new Jellyfin API client
func NewClient(config models.Config) (*Client, error) {
	client := &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	return client, client.ParseUserId()
}

func (c *Client) ParseUserId() error {
	endpoint := "/Users"

	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result []struct {
		Name string `json:"Name"`
		ID   string `json:"Id"`
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	for _, user := range result {
		if strings.ToLower(user.Name) == strings.ToLower(c.config.UserName) {
			c.config.UserID = user.ID
			return nil
		}
	}
	return fmt.Errorf("user not found: %s", c.config.UserName)
}

// GetConfig returns the client configuration
func (c *Client) GetConfig() models.Config {
	return c.config
}

// makeRequest performs an authenticated request to Jellyfin API
func (c *Client) makeRequest(method, endpoint string, body io.Reader) (*http.Response, error) {
	reqURL := c.config.ServerURL + endpoint
	req, err := http.NewRequest(method, reqURL, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Use the official Authorization header format preferred by Jellyfin
	authHeader := fmt.Sprintf("MediaBrowser Client=\"Jellyfin Manager\", Device=\"Go Client\", DeviceId=\"TraceID\", Version=\"1.0.0\", Token=\"%s\"", c.config.APIKey)
	req.Header.Set("Authorization", authHeader)

	// Fallback/Legacy header (optional, but good for compatibility)
	req.Header.Set("X-Emby-Token", c.config.APIKey)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		output, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(output))
	}

	return resp, nil
}

// GetWatchedItems retrieves all watched items from Jellyfin
func (c *Client) GetWatchedItems() ([]models.WatchedItem, error) {
	endpoint := fmt.Sprintf("/Items?userId=%s&Filters=IsPlayed&Recursive=true&IncludeItemTypes=Movie,Episode&Fields=Path,ProviderIds,SeriesName,SeasonName",
		c.config.UserID)

	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Items []struct {
			ID          string            `json:"Id"`
			Name        string            `json:"Name"`
			Type        string            `json:"Type"`
			Path        string            `json:"Path"`
			ProviderIds map[string]string `json:"ProviderIds"`
			SeriesName  string            `json:"SeriesName"`
			SeasonName  string            `json:"SeasonName"`
			UserData    struct {
				PlayedDate time.Time `json:"LastPlayedDate"`
			} `json:"UserData"`
		} `json:"Items"`
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	watchedItems := make([]models.WatchedItem, 0, len(result.Items))
	for _, item := range result.Items {
		var typeItem int
		switch item.Type {
		case "Episode":
			typeItem = models.TypeEpisode
		case "Movie":
			typeItem = models.TypeMovie
		default:
			typeItem = models.TypeUnknown
		}

		wi := models.WatchedItem{
			ID:          item.ID,
			Name:        item.Name,
			Type:        typeItem,
			PlayedDate:  item.UserData.PlayedDate,
			ProviderIDs: item.ProviderIds,
			SeriesName:  item.SeriesName,
			SeasonName:  item.SeasonName,
		}
		watchedItems = append(watchedItems, wi)
	}

	return watchedItems, nil
}

// getSeasonName retrieves the season name for an episode
func (c *Client) getSeasonName(episodeID string) (string, error) {
	endpoint := fmt.Sprintf("/Items/%s?userId=%s", episodeID, c.config.UserID)

	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		SeasonName string `json:"SeasonName"`
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	return result.SeasonName, err
}

// MarkAsWatched marks an item as watched
func (c *Client) MarkAsWatched(itemID string) error {
	endpoint := fmt.Sprintf("/UserPlayedItems/%s?userId=%s", itemID, c.config.UserID)

	resp, err := c.makeRequest("POST", endpoint, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// FindSeriesID finds the Jellyfin ID for a series by name
func (c *Client) FindSeriesID(seriesName string) (string, error) {
	endpoint := fmt.Sprintf("/Items?userId=%s&SearchTerm=%s&IncludeItemTypes=Series&Recursive=true&Limit=10",
		c.config.UserID, url.QueryEscape(seriesName))

	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Items []struct {
			ID   string `json:"Id"`
			Name string `json:"Name"`
		} `json:"Items"`
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", fmt.Errorf("decoding series search: %w", err)
	}

	// Find an exact match
	for _, item := range result.Items {
		if item.Name == seriesName {
			return item.ID, nil
		}
	}

	// If no exact match, return the first result if available
	if len(result.Items) > 0 {
		return result.Items[0].ID, nil
	}

	return "", fmt.Errorf("series not found: %s", seriesName)
}

// GetAllSeries retrieves all series from Jellyfin
func (c *Client) GetAllSeries() ([]SeriesInfo, error) {
	endpoint := fmt.Sprintf("/Items?userId=%s&Recursive=true&IncludeItemTypes=Series&Fields=ProviderIds", c.config.UserID)

	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Items []struct {
			ID          string            `json:"Id"`
			Name        string            `json:"Name"`
			ProviderIds map[string]string `json:"ProviderIds"`
		} `json:"Items"`
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("decoding series response: %w", err)
	}

	series := make([]SeriesInfo, len(result.Items))
	for i, item := range result.Items {
		series[i] = SeriesInfo{
			ID:          item.ID,
			Name:        item.Name,
			ProviderIDs: item.ProviderIds,
		}
	}

	return series, nil
}

// GetEpisodesForSeries retrieves all episodes for a series
func (c *Client) GetEpisodesForSeries(seriesID string) ([]EpisodeInfo, error) {
	endpoint := fmt.Sprintf("/Items?userId=%s&ParentId=%s&Recursive=true&IncludeItemTypes=Episode&Fields=ProviderIds,SeriesName,SeasonName,UserData",
		c.config.UserID, seriesID)

	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Items []struct {
			ID                string            `json:"Id"`
			Name              string            `json:"Name"`
			SeriesName        string            `json:"SeriesName"`
			SeasonName        string            `json:"SeasonName"`
			IndexNumber       int               `json:"IndexNumber"`
			ParentIndexNumber int               `json:"ParentIndexNumber"`
			RuntimeTicks      int64             `json:"RunTimeTicks"`
			ProviderIds       map[string]string `json:"ProviderIds"`
			UserData          struct {
				Played bool `json:"Played"`
			} `json:"UserData"`
		} `json:"Items"`
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("decoding episodes: %w", err)
	}

	episodes := make([]EpisodeInfo, len(result.Items))
	for i, item := range result.Items {
		episodes[i] = EpisodeInfo{
			ID:             item.ID,
			Name:           item.Name,
			SeriesName:     item.SeriesName,
			SeasonName:     item.SeasonName,
			SeasonNumber:   item.ParentIndexNumber,
			EpisodeNumber:  item.IndexNumber,
			RuntimeMinutes: int(item.RuntimeTicks / (60 * 10 * 1000 * 1000)),
			ProviderIDs:    item.ProviderIds,
			Played:         item.UserData.Played,
		}
	}

	return episodes, nil
}

// SeriesInfo represents basic series information
type SeriesInfo struct {
	ID          string
	Name        string
	ProviderIDs map[string]string
}

// EpisodeInfo represents episode information
type EpisodeInfo struct {
	ID             string
	Name           string
	SeriesName     string
	SeasonName     string
	SeasonNumber   int
	EpisodeNumber  int
	RuntimeMinutes int
	ProviderIDs    map[string]string
	Played         bool
}

// MovieInfo represents movie information with watched status
type MovieInfo struct {
	ID     string
	Played bool
}

// GetAllMovies retrieves all movies with their watched status
func (c *Client) GetAllMovies() (map[string]MovieInfo, map[string]MovieInfo, error) {
	endpoint := fmt.Sprintf("/Items?userId=%s&Recursive=true&IncludeItemTypes=Movie&Fields=ProviderIds,UserData", c.config.UserID)

	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Items []struct {
			ID          string            `json:"Id"`
			Name        string            `json:"Name"`
			ProviderIds map[string]string `json:"ProviderIds"`
			UserData    struct {
				Played bool `json:"Played"`
			} `json:"UserData"`
		} `json:"Items"`
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, nil, fmt.Errorf("decoding movies response: %w", err)
	}

	providerIdMap := make(map[string]MovieInfo)
	nameMap := make(map[string]MovieInfo)

	for _, item := range result.Items {
		info := MovieInfo{
			ID:     item.ID,
			Played: item.UserData.Played,
		}

		nameMap[item.Name] = info

		for provider, id := range item.ProviderIds {
			key := provider + ":" + id
			providerIdMap[key] = info
		}
	}

	return providerIdMap, nameMap, nil
}
