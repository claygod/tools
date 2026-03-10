package circle_sector_control

// Circle sector control
// Tests 1
// Copyright © 2026 Eduard Sesigin. All rights reserved. Contacts: <claygod@yandex.ru>

import (
	"sync/atomic"
)

const (
	StatusCreated int64 = iota
	StatusStarted
	StatusStopped
	StatusClosed
)

type CircleSectorControl struct {
	cap  int64
	_    [56]byte // cap в отдельной cache line
	head int64
	_    [56]byte // Cache line padding
	tail int64
	_    [56]byte // Cache line padding
}

func NewCircleSectorControl(cap int64) *CircleSectorControl {
	if cap <= 0 {
		panic("cap must be positive")
	}

	return &CircleSectorControl{
		cap:  cap,
		head: 0,
		tail: 0,
	}
}

func (c *CircleSectorControl) HeadForward() (int64, bool) {
	for {
		curTail := atomic.LoadInt64(&c.tail)
		curHead := atomic.LoadInt64(&c.head)
		newHead := c.increment(curHead)

		if newHead == curTail {
			return curHead, false
		}

		if !atomic.CompareAndSwapInt64(&c.head, curHead, newHead) {
			continue
		}

		return newHead, true
	}
}

func (c *CircleSectorControl) TailForward() (int64, bool) {
	for {
		curTail := atomic.LoadInt64(&c.tail)
		curHead := atomic.LoadInt64(&c.head)
		newTail := c.increment(curTail)

		if curHead == curTail {
			return curTail, false
		}

		if !atomic.CompareAndSwapInt64(&c.tail, curTail, newTail) {
			continue
		}

		return newTail, true
	}
}

func (c *CircleSectorControl) Head() int64 {
	return atomic.LoadInt64(&c.head)
}

func (c *CircleSectorControl) Tail() int64 {
	return atomic.LoadInt64(&c.tail)
}

func (c *CircleSectorControl) IsSingleSector() bool {
	return atomic.LoadInt64(&c.tail) == atomic.LoadInt64(&c.head)
}

func (c *CircleSectorControl) increment(in int64) int64 {
	in++

	if in == c.cap {
		in = 0
	}

	return in
}

func (c *CircleSectorControl) Utilization() float64 {
	curHead := atomic.LoadInt64(&c.head)
	curTail := atomic.LoadInt64(&c.tail)

	var used int64
	if curHead >= curTail {
		used = curHead - curTail + 1
	} else {
		used = c.cap - curTail + curHead + 1
	}

	// Защита от >100% (на случай race condition)
	if used > c.cap {
		used = c.cap
	}
	if used < 0 {
		used = 0
	}

	return float64(used) / float64(c.cap) * 100
}
