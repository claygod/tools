package batcher

// Batcher
// Indicator
// Copyright © 2018 Eduard Sesigin. All rights reserved. Contacts: <claygod@yandex.ru>

import (
	"sync/atomic"
)

const cleanerShift uint8 = 128 // shift to exclude race condition

/*
Indicator - the closing of the channels signals the completed task.
*/
type Indicator struct {
	chDone [256]chan struct{}
	cursor uint32
}

/*
NewIndicator - create new Indicator.
*/
func NewIndicator() *Indicator {
	i := &Indicator{}
	for u := 0; u < 256; u++ {
		i.chDone[u] = make(chan struct{})
	}
	return i
}

/*
SwitchChan - switch channels:
	- a new channel is created
	- the pointer switches to a new channel
	- the old channel (with a shift) is closed
*/
func (i *Indicator) SwitchChan() {
	cursor := uint8(atomic.LoadUint32(&i.cursor))
	i.chDone[cursor+1] = make(chan struct{})
	atomic.StoreUint32(&i.cursor, uint32(cursor+1))
	close(i.chDone[cursor-cleanerShift])
}

/*
GetChan - get current channel.
*/
func (i *Indicator) GetChan() chan struct{} {
	cursor := uint8(atomic.LoadUint32(&i.cursor))
	return i.chDone[cursor]
}
