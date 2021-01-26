package main

import (
	"sync"
	"sync/atomic"
)

type ProcessMap struct {
	count uint64

	syncMap sync.Map
}

func (processMap *ProcessMap) Get(key string) (*Process, bool) {
	process, ok := processMap.syncMap.Load(key)
	if !ok {
		return nil, false
	}
	return process.(*Process), ok
}

func (processMap *ProcessMap) Add(key string, value *Process) bool {
	_, ok := processMap.Get(key)
	if !ok {
		processMap.syncMap.Store(key, value)
		atomic.AddUint64(&(processMap.count), 1)
		return true
	}
	return false
}

func (processMap *ProcessMap) Delete(key string) {
	processMap.syncMap.Delete(key)
	atomic.AddUint64(&(processMap.count), ^uint64(0))
}

func (processMap *ProcessMap) Range(f func(key string, value *Process) bool) {
	fn := func(key, value interface{}) bool {
		keyString := key.(string)
		valueString := value.(*Process)

		return f(keyString, valueString)
	}

	processMap.syncMap.Range(fn)
}

func (processMap *ProcessMap) Length() int {
	return int(atomic.LoadUint64(&(processMap.count)))
}
