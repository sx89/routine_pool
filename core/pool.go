package core

import (
"errors"
"runtime"
"sync"
"time"
)

const (
	stateCreate   = 0
	stateRunning  = 1
	stateStopping = 2
	stateShutdown = 3
)

var _defaultConfig = &PoolConfig{
	MaxWorkers:     1024,
	MaxIdleWorkers: 512,
	MinIdleWorkers: 256,
	KeepAlive:      30 * time.Second,
}

// PoolConfig .
type PoolConfig struct {
	MaxWorkers     uint64
	MaxIdleWorkers uint64
	MinIdleWorkers uint64
	KeepAlive      time.Duration
}

// Pool .
type Pool struct {
	conf       *PoolConfig
	ready      *ringBuffer
	curWorkers uint64
	lock       sync.Mutex
	state      uint8
	stop       chan uint8
}

// worker .
type worker struct {
	id          uint64
	lastUseTime time.Time
	ftch        chan *FutureTask
}

var wChanCap = func() int {
	// Use blocking worker if GOMAXPROCS=1.
	// This immediately switches Serve to WorkerFunc, which results
	// in higher performance (under go1.5 at least).
	if runtime.GOMAXPROCS(0) == 1 {
		return 0
	}

	// Use non-blocking worker if GOMAXPROCS>1,
	// since otherwise the Serve caller (Acceptor) may lag accepting
	// new task if WorkerFunc is CPU-bound.
	return 1
}()
