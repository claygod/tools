package chbqueue

// Chanenel-based queue
// Copyright © 2026 Eduard Sesigin. All rights reserved. Contacts: <claygod@yandex.ru>

import (
	"errors"
	"sync"
	"sync/atomic"
)

const (
	chBQueueOpened int64 = 0
	chBQueueClosed int64 = 1
)

type ChBQueue struct {
	m           sync.Mutex
	counter     int64 // NOTE: temporary value
	closed      int64
	runned      int64
	chsListNorm chsList
	chsListHi   chsList
	outCh       chan int
}

func New() *ChBQueue {
	return nil
}

func (c *ChBQueue) Push(in int) error {
	if atomic.LoadInt64(&c.closed) != chBQueueOpened {
		return errors.New("Queue is closed")
	}

	atomic.AddInt64(&c.counter, 1)
	c.chsListNorm.add(in)

	return nil
}

func (c *ChBQueue) PushHi(in int) error {
	if atomic.LoadInt64(&c.closed) != chBQueueOpened {
		return errors.New("Queue is closed")
	}

	atomic.AddInt64(&c.counter, 1)
	c.chsListHi.add(in)

	return nil
}

func (c *ChBQueue) Pop() (int, error) {
	if atomic.LoadInt64(&c.counter) == 0 {
		if atomic.LoadInt64(&c.closed) == chBQueueClosed {
			return 0, errors.New("Queue is empty and closed")
		}

		return 0, errors.New("Queue is empty")
	}

	select {
	case val, ok := <-c.outCh:
		if !ok {
			return 0, errors.New("Queue is closed")
		}

		atomic.AddInt64(&c.counter, -1)

		return val, nil

	default:
		return 0, errors.New("Queue is empty")
	}

	return 0, nil
}

func (c *ChBQueue) Close() {
	atomic.StoreInt64(&c.closed, chBQueueClosed)

	c.chsListNorm.close()
	c.chsListHi.close()

	// TODO: work jobs and stop
}

func (c *ChBQueue) IsClosed() bool {
	return atomic.LoadInt64(&c.closed) == chBQueueClosed
}

func (c *ChBQueue) worker() {
	for stopNorm, stopHi := false, false; !stopNorm && !stopHi; {
		select {
		case val, ok := <-c.chsListHi.outCh:
			if !ok {
				stopHi = true
			} else {
				c.outCh <- val
			}

		default:
			select {
			case val, ok := <-c.chsListHi.outCh:
				if !ok {
					stopHi = true
				} else {
					c.outCh <- val
				}

			case val, ok := <-c.chsListNorm.outCh:
				if !ok {
					stopNorm = true
				} else {
					c.outCh <- val
				}
			}
		}
	}

	close(c.outCh)
}

// type queue struct {
// 	normList chsList
// 	hiList   chsList
// }

// func (q *queue) add(in int) {

// }

type chsList struct {
	list     [256]chan int
	shiftIn  uint8
	shiftOut uint8
	inCh     chan int
	outCh    chan int
	closedCh chan struct{}
	chCap    int
	closed   int64
}

func (c *chsList) add(in int) error {
	if atomic.LoadInt64(&c.closed) == chBQueueClosed {
		return errors.New("chsList is closed")
	}

	c.inCh <- in // Blocked in exceptional cases
	return nil
}

func (c *chsList) workerIn() {
	for {
		if atomic.LoadInt64(&c.closed) == chBQueueClosed {
			break
		}

		val, ok := <-c.inCh
		if !ok {
			atomic.StoreInt64(&c.closed, chBQueueClosed)

			break // case channel close
		}

		select {
		case c.list[c.shiftIn] <- val:
			// ok

		default:
			// add nov ch
			if c.shiftIn+1 != c.shiftOut {
				c.shiftIn++
				c.list[c.shiftIn] = make(chan int, c.chCap)
			}
		}
	}
}

func (c *chsList) workerOut() {
	for {
		val, ok := <-c.list[c.shiftOut]
		if !ok {
			break // case channel close
		}

		c.outCh <- val

		// closed
		if atomic.LoadInt64(&c.closed) == chBQueueClosed {
			close(c.outCh)

			break
		}

		// how shift?
		if len(c.list[c.shiftOut]) == 0 && c.shiftOut != c.shiftIn {
			c.list[c.shiftOut] = nil
			c.shiftOut++
		}
	}
}

func (c *chsList) close() {
	if atomic.LoadInt64(&c.closed) == chBQueueClosed {
		return
	}

	atomic.StoreInt64(&c.closed, chBQueueClosed)
	close(c.inCh)
	<-c.closedCh
}
