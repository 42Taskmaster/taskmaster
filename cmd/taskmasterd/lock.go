package main

import "os"

const lockFilePath = "./taskmasterd.lock"

func lockFileExists() bool {
	_, err := os.Stat(lockFilePath)
	if err == nil {
		return true
	}
	return !os.IsNotExist(err)
}

func lockFileCreate() {
	os.Create(lockFilePath)
}

func lockFileRemove() {
	os.Remove(lockFilePath)
}
