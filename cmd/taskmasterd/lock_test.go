package main

import "testing"

func TestGetLockFilePathIsInIdempotent(t *testing.T) {
	const iterations = 100

	var lastLockFilePath string

	for index := 0; index < iterations; index++ {
		if index == 0 {
			lastLockFilePath = getLockFilePath()
			continue
		}

		if newLockFilePath := getLockFilePath(); newLockFilePath != lastLockFilePath {
			t.Fatalf(
				"unexpected value returned %v; expected getLockFilePath to be idempotent and return %v",
				newLockFilePath,
				lastLockFilePath,
			)
		}
	}
}
