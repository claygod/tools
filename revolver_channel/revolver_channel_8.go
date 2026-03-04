package revolver_channel

// Revolver Channel
// Main
// Copyright © 2026 Eduard Sesigin. All rights reserved. Contacts: <claygod@yandex.ru>

import (
	"sync"
	"sync/atomic"
	"time"
)

const (
	limit8bit int = 1 << 8 // 256
)

type RevolverChannel8Bit[T any] struct {
	mu       sync.Mutex
	list     [limit8bit]chan T
	shiftIn  uint8
	shiftOut uint8
	In       chan T
	Out      chan T
	chCap    int
	status   int64
	counter  int64
	stopCh   chan struct{}
	closeCh  chan struct{}
}

func NewRevolverChannel8Bit[T any](chCap int) *RevolverChannel8Bit[T] {
	var list [limit8bit]chan T
	list[0] = make(chan T, chCap)

	rCh := &RevolverChannel8Bit[T]{
		list:     list,
		shiftIn:  0,
		shiftOut: 0,
		In:       make(chan T),
		Out:      make(chan T),
		chCap:    chCap,
		status:   StatusCreated,
		stopCh:   make(chan struct{}),
		closeCh:  make(chan struct{}),
	}

	rCh.start()

	return rCh
}

func (r *RevolverChannel8Bit[T]) workerIn() {
	for {
		val, ok := <-r.In
		if !ok {
			break // channel closed
		}

		atomic.AddInt64(&r.counter, 1)

		select {
		case r.list[r.shiftIn] <- val:
			// fmt.Println("ok", val)

		default:
			// add nov ch
			if r.shiftIn+1 != r.shiftOut {
				r.mu.Lock()
				if r.shiftIn+1 != r.shiftOut {
					r.shiftIn++
					r.list[r.shiftIn] = make(chan T, r.chCap)
				}
				r.mu.Unlock()
			}

			r.list[r.shiftIn] <- val // the case when the blocking will occur
		}
	}
}

func (r *RevolverChannel8Bit[T]) wStop() {
	<-r.stopCh

	if r.shiftOut == r.shiftIn && r.Len() == 0 {
		close(r.list[r.shiftOut])
	}
}

func (r *RevolverChannel8Bit[T]) workerOut() {
	for {
		// is closed
		if r.IsStoped() && r.shiftOut == r.shiftIn && r.Len() == 0 {

			break
		}

		val, ok := <-r.list[r.shiftOut]
		if !ok {
			break // case channel close
		}

		r.Out <- val
		atomic.AddInt64(&r.counter, -1)

		if len(r.list[r.shiftOut]) == 0 && r.shiftOut != r.shiftIn {
			r.mu.Lock()
			if len(r.list[r.shiftOut]) == 0 && r.shiftOut != r.shiftIn {
				r.list[r.shiftOut] = nil
				r.shiftOut++
			}
			r.mu.Unlock()
		}
	}

	close(r.Out)
	close(r.closeCh)
	atomic.StoreInt64(&r.status, StatusClosed)
}

func (r *RevolverChannel8Bit[T]) Len() int64 {
	return atomic.LoadInt64(&r.counter)
}

func (r *RevolverChannel8Bit[T]) start() {
	atomic.StoreInt64(&r.status, StatusStarted)

	go r.wStop()
	go r.workerIn()
	go r.workerOut()

	time.Sleep(time.Millisecond)
}

func (r *RevolverChannel8Bit[T]) Stop() {
	switch atomic.LoadInt64(&r.status) {
	case StatusCreated:
		return

	case StatusStarted:
		atomic.StoreInt64(&r.status, StatusStopped)
		close(r.In)
		close(r.stopCh)

		return

	case StatusStopped:
		return

	case StatusClosed:
		return
	}
}

func (r *RevolverChannel8Bit[T]) IsStoped() bool {
	return atomic.LoadInt64(&r.status) >= StatusStopped
}

func (r *RevolverChannel8Bit[T]) IsClosed() bool {
	return atomic.LoadInt64(&r.status) == StatusClosed
}

func (r *RevolverChannel8Bit[T]) WaitClose() {
	<-r.closeCh
}

func (r *RevolverChannel8Bit[T]) Utilization() float64 {
	r.mu.Lock()
	defer r.mu.Unlock()

	delta := r.shiftIn - r.shiftOut

	if delta == 0 {
		delta = 1
	}

	return float64(delta) / float64(limit8bit) * 100
}
