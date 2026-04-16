package backup

import (
	"errors"
	"fmt"
	"os"
	"time"
)

// Backup renames dir to dir_backup_<unix-timestamp> and returns the new path.
// Returns ("", nil) when dir does not exist — idempotent by design.
func Backup(dir string) (string, error) {
	_, err := os.Lstat(dir)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	dest := fmt.Sprintf("%s_backup_%d", dir, time.Now().Unix())
	if err := os.Rename(dir, dest); err != nil {
		return "", err
	}
	return dest, nil
}
