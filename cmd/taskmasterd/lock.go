package main

import (
	"log"
	"os"
	"path"
)

func getLockFilePath() string {
	const lockFileName = "taskmasterd.lock"

	tmpDir := os.TempDir()

	return path.Join(tmpDir, lockFileName)
}

func lockFileExists() bool {
	lockFilePath := getLockFilePath()
	log.Println("lock file path", lockFilePath)

	_, err := os.Stat(lockFilePath)
	if err == nil {
		return true
	}
	return !os.IsNotExist(err)
}

func lockFileCreate() {
	lockFilePath := getLockFilePath()
	log.Println("lock file path", lockFilePath)

	os.Create(lockFilePath)
}

func lockFileRemove() {
	lockFilePath := getLockFilePath()
	log.Println("lock file path", lockFilePath)

	os.Remove(lockFilePath)
}
