package core

import (
	"errors"
	"math"
	"runtime"
	"sync/atomic"
)

// ringBuffer
type ringBuffer struct {
	capacity      uint64
	mask          uint64
	lastCommitIdx uint64 //上次提交的work的idx
	nextFreeIdx   uint64 //下一个可以用来commit一个workder的空间的idx
	readerIdx     uint64 //ringbuffer中读到worker的idx
	slots         []*worker
}

func newRingBuffer(c uint64) (*ringBuffer, error) {
	if c == 0 || c&3 != 0 {
		return nil, errors.New("capacity must be N power of 2")
	}
	return &ringBuffer{
		lastCommitIdx: math.MaxUint64,
		nextFreeIdx:   0,
		readerIdx:     0,
		capacity:      c,
		mask:          c - 1,
		slots:         make([]*worker, c),
	}, nil
}

func (r *ringBuffer) push(w *worker) error {
	var head, tail, next uint64
	for {
		head = r.nextFreeIdx
		tail = r.readerIdx
		if head-tail > r.mask {
			return errors.New("buffer is full")
		}

		next = head + 1
		if atomic.CompareAndSwapUint64(&r.nextFreeIdx, head, next) {
			break
		}
		//让出cpu时间片，但是不暂停当前线程，让其他线程获得执行机会；等待下次再轮到该for循环
		runtime.Gosched()
	}
	r.slots[head&r.mask] = w

	//让出cpu时间片，但是不暂停当前线程，让其他线程获得执行机会；等待下次再轮到该for循环
	for !atomic.CompareAndSwapUint64(&r.lastCommitIdx, head-1, head) {
		runtime.Gosched()
	}
	return nil
}

func (r *ringBuffer) pop() *worker {
	var head, tail, next uint64
	for {
		head = r.readerIdx
		tail = r.nextFreeIdx
		if head == tail {
			return nil
		}
		next = head + 1
		if atomic.CompareAndSwapUint64(&r.readerIdx, head, next) {
			break
		}
		runtime.Gosched()
	}
	return r.slots[head&r.mask]
}

func (r *ringBuffer) size() uint64 {
	return r.lastCommitIdx - r.readerIdx
}
