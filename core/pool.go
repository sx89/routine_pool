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

func newWorker(wid uint64) *worker {
	return &worker{
		id:          wid,
		lastUseTime: time.Now(),
		ftch:        make(chan *FutureTask, wChanCap),
	}
}

// NewWorkerPool .
func NewWorkerPool(capacity uint64, conf *PoolConfig) (p *Pool, err error) {
	if capacity == 0 || capacity&3 != 0 {
		err = errors.New("capacity must bigger than zero and N power of 2")
		return
	}

	rb, err := newRingBuffer(capacity)
	if err != nil {
		return
	}
	if conf == nil {
		conf = _defaultConfig
	}
	conf.MaxWorkers = capacity
	p = &Pool{
		conf:       conf,
		ready:      rb,
		curWorkers: 0,
		state:      stateCreate,
		stop:       make(chan uint8, 1),
	}
	return
}


func (p *Pool) changeState(old, new uint8) bool {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.state != old {
		return false
	}

	p.state = new
	return true
}

// Start .
func (p *Pool) Start() error {
	if !p.changeState(stateCreate, stateRunning) {
		return errors.New("workerpool already started")
	}
	go func() {
		defer close(p.stop)
		for {
			p.clean()
			select {
			case <-p.stop:
				p.cleanAll()
				for !p.changeState(stateStopping, stateShutdown) {
					runtime.Gosched()
				}
				return
			default:
				time.Sleep(p.conf.KeepAlive)
			}
		}
	}()
	return nil
}