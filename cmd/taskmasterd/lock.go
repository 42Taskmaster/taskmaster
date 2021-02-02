package main

import (
	"os"
	"path"
)

func joinTempDir(file string) string {
	tmpDir := os.TempDir()

	return path.Join(tmpDir, file)
}

func getLockFilePath() string {
	const lockFileName = "taskmasterd.lock"

	return joinTempDir(lockFileName)
}

func lockFileExists() bool {
	lockFilePath := getLockFilePath()

	_, err := os.Stat(lockFilePath)
	if err == nil {
		return true
	}
	return !os.IsNotExist(err)
}

func lockFileCreate() {
	lockFilePath := getLockFilePath()

	os.Create(lockFilePath)
}

func lockFileRemove() {
	lockFilePath := getLockFilePath()

	os.Remove(lockFilePath)
}
