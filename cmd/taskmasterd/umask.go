package main

import (
	"log"
	"strconv"
	"sync"
	"syscall"
)

var Umask = -1
var UmaskLock sync.Mutex

func SetUmask(umask string) {
	if len(umask) == 0 {
		return
	}

	UmaskLock.Lock()
	defer UmaskLock.Unlock()

	octal, err := strconv.ParseInt(umask, 8, 64)
	if err != nil {
		log.Panic(err)
	}

	Umask = syscall.Umask(int(octal))
}

func ResetUmask() {
	UmaskLock.Lock()
	defer UmaskLock.Unlock()
	if Umask != -1 {
		syscall.Umask(Umask)
	}
}
