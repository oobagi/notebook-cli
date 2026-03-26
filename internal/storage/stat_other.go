//go:build !darwin

package storage

import (
	"os"
	"time"
)

func fileCreatedAt(info os.FileInfo) time.Time {
	return info.ModTime() // fallback: no birthtime on Linux
}
