package models

import "time"

// WatchedItem represents a watched movie or episode
type WatchedItem struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Type        int               `json:"type"`
	SeriesName  string            `json:"series_name,omitempty"`
	SeasonName  string            `json:"season_name,omitempty"`
	PlayedDate  time.Time         `json:"played_date"`
	ProviderIDs map[string]string `json:"provider_ids,omitempty"`
}

const (
	// TypeUnknown is set if the item is neither a movie nor an episode (which shouldn't happen)
	TypeUnknown = iota
	// TypeMovie is set if the item is a movie
	TypeMovie
	// TypeEpisode is set if the item is an episode
	TypeEpisode
)

// Backup holds all watched items
type Backup struct {
	CreatedAt    time.Time     `json:"created_at"`
	ServerURL    string        `json:"server_url"`
	UserID       string        `json:"user_id"`
	UserName     string        `json:"user_name"`
	AppVersion   string        `json:"version"`
	WatchedItems []WatchedItem `json:"watched_items"`
}

// Config holds connection settings
type Config struct {
	ServerURL string
	APIKey    string
	UserID    string
	UserName  string
}

// MissingEpisode represents an episode that exists in TVDB but not in Jellyfin
type MissingEpisode struct {
	SeriesName    string
	SeasonNumber  int
	EpisodeNumber int
	EpisodeName   string
	AirDate       string
	Overview      string
}
