package batcher

// Batcher
// Worker
// Copyright © 2018 Eduard Sesigin. All rights reserved. Contacts: <claygod@yandex.ru>

import (
	"bytes"
	//"fmt"
	"runtime"
	"sync/atomic"
	//"time"
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
		var u int
		// begin
		select {
		//		case inData := <-b.chInput:
		//			if _, err := buf.Write(inData); err != nil {
		//				b.alarm(err)
		//			}
		case <-b.chStop:
			atomic.StoreInt64(&b.stopFlag, stateStop)
			return
			// case inData := <-b.chInput:
			// 	if _, err := buf.Write(inData); err != nil {
			// 		b.alarm(err)

			// 	} else {
			// 		u++
			// 	}
		default:
			break
		}
		// batch fill
		for i := 0; i < b.batchSize; i++ { // -1
			select {
			case inData := <-b.chInput:
				if _, err := buf.Write(inData); err != nil {
					b.alarm(err)

				} else {
					u++
				}
			default:
				break
			}
		}
		// batch to out
		bOut := buf.Bytes()
		if len(bOut) > 0 {
			//fmt.Println("Текущий батч - ", u)
			if _, err := b.work.Write(buf.Bytes()); err != nil {
				atomic.StoreInt64(&b.stopFlag, stateStop)
				b.alarm(err)
				return
			}
		} else {
			//fmt.Println("Почему-то  len(bOut) == 0 ")
			//time.Sleep(10 * time.Microsecond)
			runtime.Gosched()
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

		//time.Sleep(100 * time.Millisecond)
	}
}
