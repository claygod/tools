package batcher

// Batcher
// Indicator
// Copyright © 2018 Eduard Sesigin. All rights reserved. Contacts: <claygod@yandex.ru>

import (
	"fmt"
	"sync/atomic"
)

const cleanerShift uint8 = 128 // shift to exclude race condition

/*
indicator - the closing of the channels signals the completed task.
*/
type indicator struct {
	chDone [256]chan struct{}
	cursor uint32
}

/*
newIndicator - create new Indicator.
*/
func newIndicator() *indicator {
	i := &indicator{}
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
func (i *indicator) switchChan() {
	fmt.Println("indicator switch ", uint8(atomic.LoadUint32(&i.cursor)))
	cursor := uint8(atomic.LoadUint32(&i.cursor))
	i.chDone[cursor+1] = make(chan struct{})
	atomic.StoreUint32(&i.cursor, uint32(cursor+1))
	close(i.chDone[cursor-cleanerShift])
}

/*
getChan - get current channel.
*/
func (i *indicator) getChan() chan struct{} {
	cursor := uint8(atomic.LoadUint32(&i.cursor))
	return i.chDone[cursor]
}
