package revolver_channel

// Revolver Channel
// Main
// Copyright © 2026 Eduard Sesigin. All rights reserved. Contacts: <claygod@yandex.ru>

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	csc "github.com/claygod/tools/circle_sector_control"
)

type RevolverChannel[T any] struct {
	mu      sync.Mutex
	list    []chan T
	shifter *csc.CircleSectorControl
	In      chan T
	Out     chan T
	chCap   int
	status  int64
	counter int64
	stopCh  chan struct{}
	closeCh chan struct{}
}

func NewRevolverChannel[T any](chCap int, chCount int) (*RevolverChannel[T], error) {
	if chCap <= 0 {
		return nil, fmt.Errorf("chCap must be positive (cur %d)", chCap)
	}

	if chCount <= 0 {
		return nil, fmt.Errorf("chCount must be positive (cur %d)", chCount)
	}

	list := make([]chan T, chCount)
	list[0] = make(chan T, chCap)

	rCh := &RevolverChannel[T]{
		list:    list,
		shifter: csc.NewCircleSectorControl(int64(chCount)),
		In:      make(chan T),
		Out:     make(chan T),
		chCap:   chCap,
		status:  StatusCreated,
		stopCh:  make(chan struct{}),
		closeCh: make(chan struct{}),
	}

	rCh.start()

	return rCh, nil
}

func (r *RevolverChannel[T]) workerIn() {
	for {
		val, ok := <-r.In
		if !ok {
			break // channel closed
		}

		atomic.AddInt64(&r.counter, 1)

		select {
		case r.list[r.shifter.Head()] <- val:
			// fmt.Println("ok", val)

		default:
			// add nov ch
			newShiftIn, ok := r.shifter.HeadForward()
			if ok {
				if r.list[newShiftIn] == nil {
					r.list[newShiftIn] = make(chan T, r.chCap)
				}
			}

			r.list[newShiftIn] <- val // the case when the blocking will occur
		}
	}
}

func (r *RevolverChannel[T]) wStop() {
	<-r.stopCh

	// Ждём пока все значения будут обработаны
	for {
		if r.shifter.IsSingleSector() && r.Len() == 0 {
			tail := r.shifter.Tail()
			if r.list[tail] != nil {
				close(r.list[tail]) // Это разбудит workerOut()
			}
			break
		}
		time.Sleep(time.Microsecond) // Даём время workerOut() обработать
	}
}

func (r *RevolverChannel[T]) workerOut() {
	for {
		tail := r.shifter.Tail()

		// Защита от nil канала
		if r.list[tail] == nil {
			if r.IsStoped() {
				break // Выход если остановлено
			}
			time.Sleep(time.Microsecond)
			continue
		}

		val, ok := <-r.list[tail]
		if !ok {
			break // Канал закрыт → выход
		}

		r.Out <- val
		atomic.AddInt64(&r.counter, -1)

		// Освобождаем пустые каналы
		if !r.shifter.IsSingleSector() {
			if len(r.list[tail]) == 0 {
				if _, ok := r.shifter.TailForward(); ok {
					r.list[tail] = nil
				}
			}
		}
	}

	close(r.Out)
	close(r.closeCh)
	atomic.StoreInt64(&r.status, StatusClosed)
}

func (r *RevolverChannel[T]) Len() int64 {
	return atomic.LoadInt64(&r.counter)
}

func (r *RevolverChannel[T]) start() {
	atomic.StoreInt64(&r.status, StatusStarted)

	go r.wStop()
	go r.workerIn()
	go r.workerOut()

	time.Sleep(time.Millisecond)
}

func (r *RevolverChannel[T]) Stop() {
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

func (r *RevolverChannel[T]) IsStoped() bool {
	return atomic.LoadInt64(&r.status) >= StatusStopped
}

func (r *RevolverChannel[T]) IsClosed() bool {
	return atomic.LoadInt64(&r.status) == StatusClosed
}

func (r *RevolverChannel[T]) WaitClose() {
	<-r.closeCh
}

func (r *RevolverChannel[T]) Utilization() float64 {
	return r.shifter.Utilization()
}
