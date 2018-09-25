package batcher

// Batcher
// Worker
// Copyright © 2018 Eduard Sesigin. All rights reserved. Contacts: <claygod@yandex.ru>

import (
	"bytes"
	"sync/atomic"
)

/*
worker - basic cycle.

	- creates a batch
	- passes the batch to the vryter
	- check if you need to stop
	- switches the channel
	- zeroes the buffer under the new batch
*/
func (b *Batcher) worker() {
	var buf bytes.Buffer
	for {
		// begin
		select {
		case inData := <-b.chInput:
			buf.Write(inData)
			if _, err := buf.Write(inData); err != nil {
				b.alarm(err)
			}
		case <-b.chStop:
			atomic.StoreInt64(&b.stopFlag, stateStop)
			return
		}
		// batch fill
		for i := 0; i < b.batchSize-1; i++ {
			select {
			case inData := <-b.chInput:
				buf.Write(inData)
				if _, err := buf.Write(inData); err != nil {
					b.alarm(err)
				}
			default:
				break
			}
		}
		// batch to out
		if _, err := b.work.Write(buf.Bytes()); err != nil {
			atomic.StoreInt64(&b.stopFlag, stateStop)
			b.alarm(err)
			return
		}
		// exit-check
		select {
		case <-b.chStop:
			atomic.StoreInt64(&b.stopFlag, stateStop)
			return
		default:
		}
		// cursor (indicator)  switch
		b.indicator.switchChan()
		buf.Reset()
	}
}
