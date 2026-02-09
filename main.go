package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/forceu/jellyfinmanager/api/jellyfin"
	"github.com/forceu/jellyfinmanager/api/tvdb"
	"github.com/forceu/jellyfinmanager/environment"
	"github.com/forceu/jellyfinmanager/models"
)

const (
	appVersion = "1.0.0"
)

func main() {
	// Command-line flags
	var (
		serverURL       = flag.String("server", "", "Jellyfin server URL (e.g., http://localhost:8096)")
		apiKey          = flag.String("apikey", "", "Jellyfin API key")
		userName        = flag.String("user", "", "Jellyfin user name")
		tvdbAPIKey      = flag.String("tvdb-apikey", "", "TVDB API key (for missing episodes check)")
		backupFile      = flag.String("file", environment.DefaultBackupFile, "Backup file path")
		backup          = flag.Bool("backup", false, "Perform backup")
		restore         = flag.Bool("restore", false, "Perform restore")
		findMissing     = flag.Bool("find-missing", false, "Find missing episodes using TVDB")
		includeSpecials = flag.Bool("include-specials", false, "Include special episodes in missing episode check")
	)

	flag.Parse()

	// Validate required flags
	if *serverURL == "" {
		*serverURL = os.Getenv("JELLYFIN_SERVER")
	}
	if *apiKey == "" {
		*apiKey = os.Getenv("JELLYFIN_API_KEY")
	}
	if *userName == "" {
		*userName = os.Getenv("JELLYFIN_USER")
	}
	if *tvdbAPIKey == "" {
		*tvdbAPIKey = os.Getenv("TVDB_API_KEY")
	}

	if *serverURL == "" || *apiKey == "" || *userName == "" {
		fmt.Println("Error: Missing required configuration")
		fmt.Println("\nUsage:")
		fmt.Println("  Backup:        jellyfinmanager -backup -server URL -apikey KEY -user NAME [-file backup.json]")
		fmt.Println("  Restore:       jellyfinmanager -restore -server URL -apikey KEY -user NAME [-file backup.json]")
		fmt.Println("  Find Missing:  jellyfinmanager -find-missing -server URL -apikey KEY -user NAME -tvdb-apikey KEY [-include-specials]")
		fmt.Println("\nOr set environment variables:")
		fmt.Println("  JELLYFIN_SERVER, JELLYFIN_API_KEY, JELLYFIN_USER, TVDB_API_KEY")
		os.Exit(1)
	}

	config := models.Config{
		ServerURL: strings.TrimSuffix(*serverURL, "/"),
		APIKey:    *apiKey,
		UserName:  *userName,
	}

	client, err := jellyfin.NewClient(config)
	if err != nil {
		fmt.Printf("Error logging in to Jellyfin: %v\n", err)
		os.Exit(1)
	}

	// Execute requested operation
	if *backup {
		err = performBackup(client, *backupFile)
		if err != nil {
			fmt.Printf("Backup failed: %v\n", err)
			os.Exit(1)
		}
	} else if *restore {
		err = performRestore(client, *backupFile)
		if err != nil {
			fmt.Printf("Restore failed: %v\n", err)
			os.Exit(1)
		}
	} else if *findMissing {
		if *tvdbAPIKey == "" {
			fmt.Println("Error: TVDB API key required for finding missing episodes")
			fmt.Println("Use -tvdb-apikey flag or set TVDB_API_KEY environment variable")
			os.Exit(1)
		}

		err = performFindMissing(client, *tvdbAPIKey, *includeSpecials)
		if err != nil {
			fmt.Printf("Find missing episodes failed: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Println("Error: Please specify -backup, -restore, or -find-missing")
		os.Exit(1)
	}
}

func performBackup(client *jellyfin.Client, filename string) error {
	fmt.Printf("Fetching watched items from Jellyfin for user %s...\n", client.GetConfig().UserName)
	watchedItems, err := client.GetWatchedItems()
	if err != nil {
		return fmt.Errorf("getting watched items: %w", err)
	}

	backup := models.Backup{
		CreatedAt:    time.Now(),
		ServerURL:    client.GetConfig().ServerURL,
		UserID:       client.GetConfig().UserID,
		UserName:     client.GetConfig().UserName,
		AppVersion:   appVersion,
		WatchedItems: watchedItems,
	}

	data, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling backup: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("writing backup file: %w", err)
	}

	fmt.Printf("✓ Backed up %d watched items to %s\n", len(watchedItems), filename)
	return nil
}

func performRestore(client *jellyfin.Client, filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("reading backup file: %w", err)
	}

	var backup models.Backup
	if err := json.Unmarshal(data, &backup); err != nil {
		return fmt.Errorf("unmarshaling backup: %w", err)
	}

	fmt.Printf("Restoring %d watched items for %s from backup created at %s\n",
		len(backup.WatchedItems), client.GetConfig().UserName, backup.CreatedAt.Format(time.RFC3339))

	// Group items by type
	movies := make([]models.WatchedItem, 0)
	tvShowMap := make(map[string]map[string][]models.WatchedItem)

	for _, item := range backup.WatchedItems {
		if item.Type == models.TypeMovie {
			movies = append(movies, item)
		} else if item.Type == models.TypeEpisode {
			if tvShowMap[item.SeriesName] == nil {
				tvShowMap[item.SeriesName] = make(map[string][]models.WatchedItem)
			}
			tvShowMap[item.SeriesName][item.SeasonName] = append(tvShowMap[item.SeriesName][item.SeasonName], item)
		}
	}

	fmt.Printf("Found %d movies and %d TV shows\n", len(movies), len(tvShowMap))

	successful := 0
	failed := 0
	total := 0

	// Process movies
	if len(movies) > 0 {
		fmt.Printf("\n=== Processing %d Movies ===\n", len(movies))
		movieSuccess, movieFailed := restoreMovies(client, movies)
		successful += movieSuccess
		failed += movieFailed
		total += len(movies)
	}

	// Process TV shows
	if len(tvShowMap) > 0 {
		fmt.Printf("\n=== Processing %d TV Shows ===\n", len(tvShowMap))
		tvSuccess, tvFailed := restoreTVShows(client, tvShowMap)
		successful += tvSuccess
		failed += tvFailed
		for _, seasons := range tvShowMap {
			for _, episodes := range seasons {
				total += len(episodes)
			}
		}
	}

	fmt.Printf("\n=== Restore Complete ===\n")
	fmt.Printf("Successful: %d\n", successful)
	fmt.Printf("Failed: %d\n", failed)
	fmt.Printf("Total: %d\n", total)

	return nil
}

func restoreMovies(client *jellyfin.Client, movies []models.WatchedItem) (successful, failed int) {
	providerIdMap, nameMap, err := client.GetAllMovies()
	if err != nil {
		fmt.Printf("Error fetching movies from server: %v\n", err)
		return 0, len(movies)
	}

	for i, movie := range movies {
		fmt.Printf("[%d/%d] Processing movie: %s\n", i+1, len(movies), movie.Name)

		var movieInfo jellyfin.MovieInfo
		found := false

		// Try provider IDs first
		for provider, id := range movie.ProviderIDs {
			key := provider + ":" + id
			if info, exists := providerIdMap[key]; exists {
				movieInfo = info
				found = true
				break
			}
		}

		// Fallback to name matching
		if !found {
			if info, exists := nameMap[movie.Name]; exists {
				movieInfo = info
				found = true
			}
		}

		if !found {
			fmt.Printf("  ✗ Could not find movie\n")
			failed++
			continue
		}

		// Skip if already watched
		if movieInfo.Played {
			fmt.Println("  ○ Already watched, skipping")
			successful++
			continue
		}

		// Mark as watched
		if err := client.MarkAsWatched(movieInfo.ID); err != nil {
			fmt.Printf("  ✗ Failed to mark as watched: %v\n", err)
			failed++
			continue
		}

		fmt.Println("  ✓ Marked as watched")
		successful++
	}

	return successful, failed
}

func restoreTVShows(client *jellyfin.Client, tvShowMap map[string]map[string][]models.WatchedItem) (successful, failed int) {
	showCount := 0
	for seriesName, seasons := range tvShowMap {
		showCount++
		episodeCount := 0
		for _, episodes := range seasons {
			episodeCount += len(episodes)
		}

		fmt.Printf("\n[%d/%d] Processing show: %s (%d episodes)\n", showCount, len(tvShowMap), seriesName, episodeCount)

		// Find the series ID
		seriesID, err := client.FindSeriesID(seriesName)
		if err != nil {
			fmt.Printf("  ✗ Error finding series: %v\n", err)
			for _, episodes := range seasons {
				failed += len(episodes)
			}
			continue
		}

		// Fetch all episodes for this series
		episodes, err := client.GetEpisodesForSeries(seriesID)
		if err != nil {
			fmt.Printf("  ✗ Error fetching episodes: %v\n", err)
			for _, eps := range seasons {
				failed += len(eps)
			}
			continue
		}

		// Build lookup maps
		type EpisodeInfo struct {
			ID     string
			Played bool
		}
		providerIdMap := make(map[string]EpisodeInfo)
		nameSeasonMap := make(map[string]EpisodeInfo)

		for _, ep := range episodes {
			info := EpisodeInfo{
				ID:     ep.ID,
				Played: ep.Played,
			}

			key := ep.SeasonName + ":" + ep.Name
			nameSeasonMap[key] = info

			for provider, id := range ep.ProviderIDs {
				providerKey := provider + ":" + id
				providerIdMap[providerKey] = info
			}
		}

		// Process each season
		for seasonName, seasonEpisodes := range seasons {
			fmt.Printf("  Season: %s (%d episodes)\n", seasonName, len(seasonEpisodes))

			for _, episode := range seasonEpisodes {
				var episodeInfo EpisodeInfo
				found := false

				// Try provider IDs first
				for provider, id := range episode.ProviderIDs {
					key := provider + ":" + id
					if info, exists := providerIdMap[key]; exists {
						episodeInfo = info
						found = true
						break
					}
				}

				// Fallback to season + name matching
				if !found {
					key := episode.SeasonName + ":" + episode.Name
					if info, exists := nameSeasonMap[key]; exists {
						episodeInfo = info
						found = true
					}
				}

				if !found {
					fmt.Printf("    ✗ %s - not found\n", episode.Name)
					failed++
					continue
				}

				// Skip if already watched
				if episodeInfo.Played {
					successful++
					continue
				}

				// Mark as watched
				if err := client.MarkAsWatched(episodeInfo.ID); err != nil {
					fmt.Printf("    ✗ %s - failed to mark: %v\n", episode.Name, err)
					failed++
					continue
				}

				successful++
			}

			fmt.Printf("    ✓ Processed %d episodes\n", len(seasonEpisodes))
		}
	}

	return successful, failed
}

func performFindMissing(jellyfinClient *jellyfin.Client, tvdbAPIKey string, includeSpecials bool) error {
	fmt.Println("Initializing TVDB client...")
	tvdbClient := tvdb.NewClient(tvdbAPIKey)

	if err := tvdbClient.Login(); err != nil {
		return fmt.Errorf("TVDB login failed: %w", err)
	}
	fmt.Println("✓ TVDB authentication successful")

	fmt.Println("\nFetching all series from Jellyfin...")
	series, err := jellyfinClient.GetAllSeries()
	if err != nil {
		return fmt.Errorf("fetching Jellyfin series: %w", err)
	}
	fmt.Printf("✓ Found %d series in Jellyfin\n", len(series))
	fmt.Println("Checking for missing episodes...")
	totalMissing := 0

	for i, s := range series {
		// Check if series has TVDB ID
		tvdbID, hasTVDB := s.ProviderIDs["Tvdb"]
		if !hasTVDB {
			continue
		}

		// Get episodes from TVDB
		tvdbEpisodes, err := tvdbClient.GetSeriesEpisodes(tvdbID)
		if err != nil {
			fmt.Printf("\n[%d/%d] %s (TVDB: %s)\n", i+1, len(series), s.Name, tvdbID)
			fmt.Printf("  ⚠ Could not fetch TVDB episodes: %v\n", err)
			continue
		}

		// Get episodes from Jellyfin
		jellyfinEpisodes, err := jellyfinClient.GetEpisodesForSeries(s.ID)
		if err != nil {
			fmt.Printf("\n[%d/%d] %s (TVDB: %s)\n", i+1, len(series), s.Name, tvdbID)
			fmt.Printf("  ⚠ Could not fetch Jellyfin episodes: %v\n", err)
			continue
		}

		// Build map of existing episodes and store runtime seconds
		// The runtime is required to check if two multi-part episodes have been merged
		existingEpisodes := make(map[string]int)
		for _, ep := range jellyfinEpisodes {
			key := fmt.Sprintf("%d:%d", ep.SeasonNumber, ep.EpisodeNumber)
			existingEpisodes[key] = ep.RuntimeMinutes
		}

		// Find missing episodes
		missing := tvdb.FindMissingEpisodes(tvdbEpisodes, existingEpisodes, includeSpecials)

		if len(missing) != 0 {
			fmt.Printf("\n[%d/%d] %s (TVDB: %s)\n", i+1, len(series), s.Name, tvdbID)
			fmt.Printf("  ⚠ Missing %d episodes (of %d total):\n", len(missing), len(tvdbEpisodes))
			for _, m := range missing {
				fmt.Printf("    - S%02dE%02d: %s (Aired: %s)\n",
					m.SeasonNumber, m.EpisodeNumber, m.EpisodeName, m.AirDate)
			}
			totalMissing += len(missing)
		}

	}

	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("Total series checked: %d\n", len(series))
	fmt.Printf("Total missing episodes: %d\n", totalMissing)

	return nil
}
