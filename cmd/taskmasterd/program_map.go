package main

import (
	"sync"
	"sync/atomic"
)

type ProgramMap struct {
	count uint64

	syncMap sync.Map
}

func (programMap *ProgramMap) Get(key string) (*Program, bool) {
	program, ok := programMap.syncMap.Load(key)
	if !ok {
		return nil, false
	}
	return program.(*Program), ok
}

func (programMap *ProgramMap) Add(key string, value *Program) bool {
	_, ok := programMap.Get(key)
	if !ok {
		programMap.syncMap.Store(key, value)
		atomic.AddUint64(&(programMap.count), 1)
		return true
	}
	return false
}

func (programMap *ProgramMap) Delete(key string) {
	programMap.syncMap.Delete(key)
	atomic.AddUint64(&(programMap.count), ^uint64(0))
}

func (programMap *ProgramMap) Range(f func(key string, value *Program) bool) {
	fn := func(key, value interface{}) bool {
		keyString := key.(string)
		valueString := value.(*Program)

		return f(keyString, valueString)
	}

	programMap.syncMap.Range(fn)
}

func (programMap *ProgramMap) Length() int {
	return int(atomic.LoadUint64(&(programMap.count)))
}
