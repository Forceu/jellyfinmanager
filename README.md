# Jellyfin Manager

A command-line tool for managing Jellyfin watched status, featuring backup/restore capabilities and missing episode detection using TVDB.

> ⚠️ **AI Disclaimer**: Portions of this codebase were generated with the assistance of AI tools.

## Features

- **Backup Watched Status**: Export all watched movies and TV episodes to a JSON file
- **Restore Watched Status**: Import watched status from a backup file to same or different user
- **Find Missing Episodes**: Compare your Jellyfin library against TVDB to identify missing episodes
- **Multi-Platform Support**: Available as a standalone binary or Docker container
- **Flexible Configuration**: Command-line flags or environment variables

## Quick Start

### Using Docker (Recommended)

```bash
# Backup watched status
docker run --rm -v $(pwd)/backup:/backup \
  ghcr.io/forceu/jellyfinmanager:latest \
  -backup \
  -server "https://your.jellyfin.server" \
  -apikey "your-api-key" \
  -user "your-username"

# Restore watched status
docker run --rm -v $(pwd)/backup:/backup \
  ghcr.io/forceu/jellyfinmanager:latest \
  -restore \
  -server "https://your.jellyfin.server" \
  -apikey "your-api-key" \
  -user "your-username"

# Find missing episodes
docker run --rm \
  ghcr.io/forceu/jellyfinmanager:latest \
  -find-missing \
  -server "https://your.jellyfin.server" \
  -apikey "your-api-key" \
  -user "your-username" \
  -tvdb-apikey "your-tvdb-api-key"
```

### Using Binary

Download the latest release for your platform and run:

```bash
# Backup
./jellyfinmanager -backup -server "http://localhost:8096" -apikey "KEY" -user "USER"

# Restore
./jellyfinmanager -restore -server "http://localhost:8096" -apikey "KEY" -user "USER"

# Find missing episodes
./jellyfinmanager -find-missing -server "http://localhost:8096" \
  -apikey "KEY" -user "USER" -tvdb-apikey "TVDB_KEY"
```

## Installation

### Docker

Pull the pre-built image:

```bash
docker pull ghcr.io/forceu/jellyfinmanager:latest
```

Or build locally:

```bash
docker build -t jellyfinmanager .
```

### From Source

Requires Go 1.25 or later:

```bash
git clone https://github.com/forceu/jellyfinmanager.git
cd jellyfinmanager
go build -o jellyfinmanager
```

## Configuration

### Command-Line Flags

| Flag | Description | Required |
|------|-------------|----------|
| `-server` | Jellyfin server URL (e.g., `http://localhost:8096`) | Yes* |
| `-apikey` | Jellyfin API key | Yes* |
| `-user` | Jellyfin username | Yes* |
| `-tvdb-apikey` | TVDB API key (required for `-find-missing`) | For find-missing |
| `-file` | Backup file path (default: `jellyfin_watched_backup.json`) | No |
| `-backup` | Perform backup operation | ** |
| `-restore` | Perform restore operation | ** |
| `-find-missing` | Find missing episodes using TVDB | ** |
| `-include-specials` | Include special episodes in missing episode check | No |

\* Can be set via environment variables  
\** One operation flag is required

### Environment Variables

Instead of command-line flags, you can use:

- `JELLYFIN_SERVER` - Jellyfin server URL
- `JELLYFIN_API_KEY` - Jellyfin API key
- `JELLYFIN_USER` - Jellyfin username
- `TVDB_API_KEY` - TVDB API key

### Getting API Keys

#### Jellyfin API Key
1. Log in to your Jellyfin server
2. Go to Dashboard → API Keys
3. Click "+" to create a new API key
4. Give it a name (e.g., "Jellyfin Manager") and save

#### TVDB API Key
1. Register at [TheTVDB](https://www.thetvdb.com/)
2. Subscribe to an API plan (free tier available)
3. Generate an API key from your account settings

## Usage Examples

### Backup Watched Status

Create a backup of all watched movies and TV episodes:

```bash
jellyfinmanager -backup \
  -server "http://localhost:8096" \
  -apikey "your-api-key" \
  -user "username" \
  -file "backup.json"
```

The backup file contains:
- Timestamp of backup creation
- Server URL and user information
- All watched items with metadata (provider IDs, names, dates)

### Restore Watched Status

Restore watched status from a backup (useful when migrating servers or users):

```bash
jellyfinmanager -restore \
  -server "http://localhost:8096" \
  -apikey "your-api-key" \
  -user "username" \
  -file "backup.json"
```

The restore process:
- Matches items using provider IDs (IMDB, TMDB, TVDB)
- Falls back to name matching if provider IDs don't match
- Skips items already marked as watched
- Provides detailed progress and summary

### Find Missing Episodes

Identify episodes that exist in TVDB but are missing from your Jellyfin library:

```bash
jellyfinmanager -find-missing \
  -server "http://localhost:8096" \
  -apikey "your-api-key" \
  -user "username" \
  -tvdb-apikey "your-tvdb-key"
```

Optional: Include special episodes (Season 0):

```bash
jellyfinmanager -find-missing \
  -server "http://localhost:8096" \
  -apikey "your-api-key" \
  -user "username" \
  -tvdb-apikey "your-tvdb-key" \
  -include-specials
```

## Docker Compose

For scheduled backups, you can use docker-compose:

```yaml
version: '3.8'

services:
  jellyfin-backup:
    image: ghcr.io/forceu/jellyfinmanager:latest
    container_name: jellyfin-backup
    volumes:
      - ./backup:/backup
    environment:
      - JELLYFIN_SERVER=http://jellyfin:8096
      - JELLYFIN_API_KEY=your-api-key
      - JELLYFIN_USER=your-username
    command: ["-backup"]
    # Optionally use a cron container to run this on schedule
```

For scheduled runs, consider using a cron container or system cron:

```bash
# Add to crontab for daily backups at 2 AM
0 2 * * * docker run --rm -v /path/to/backup:/backup -e JELLYFIN_SERVER=... ghcr.io/forceu/jellyfinmanager:latest -backup
```

## Backup File Format

The backup file is a JSON file with the following structure:

```json
{
  "created_at": "2026-02-09T15:30:00Z",
  "server_url": "http://jellyfin.example.com:8096",
  "user_id": "abc123...",
  "user_name": "username",
  "version": "1.0.0",
  "watched_items": [
    {
      "id": "item-id",
      "name": "Movie or Episode Name",
      "type": 1,
      "series_name": "TV Show Name",
      "season_name": "Season 1",
      "played_date": "2026-01-15T20:30:00Z",
      "provider_ids": {
        "Imdb": "tt1234567",
        "Tmdb": "12345",
        "Tvdb": "67890"
      }
    }
  ]
}
```

**Type values:**
- `1`: Movie
- `2`: TV Episode

## Architecture

The project is organized into the following packages:

- **main**: CLI interface and orchestration logic
- **api/jellyfin**: Jellyfin API client for fetching and updating data
- **api/tvdb**: TVDB API client for episode metadata
- **models**: Data structures and types

## Troubleshooting

### "Could not find movie/episode"

The restore process first tries to match using provider IDs (IMDB, TMDB, TVDB), then falls back to name matching. If an item can't be found:

1. Ensure the item exists in your current Jellyfin library
2. Check that metadata providers are properly configured
3. Verify the item names match between backup and current library

### "Error logging in to Jellyfin"

- Verify your server URL is correct and accessible
- Ensure the API key is valid and not expired
- Check that the username exists on the server

### "TVDB login failed"

- Verify your TVDB API key is valid
- Ensure you have an active TVDB subscription
- Check your internet connection

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the GPL License - see the LICENSE file for details.

## Acknowledgments

- Built for [Jellyfin](https://jellyfin.org/) - The Free Software Media System
- Episode metadata from [TheTVDB](https://www.thetvdb.com/)
- Parts of this code were generated with AI assistance

## Support

For bugs and feature requests, please use the [GitHub Issues](https://github.com/forceu/jellyfinmanager/issues) page.
