//go:build !docker

package environment

const IsDocker = false

const DefaultBackupFile = "jellyfin_watched_backup.json"
