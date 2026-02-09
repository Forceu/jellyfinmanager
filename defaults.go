//go:build !docker

package main

const isDocker = false

const defaultBackupFile = "jellyfin_watched_backup.json"
