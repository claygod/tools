package batcher

// Batcher
// Main
// Copyright © 2018 Eduard Sesigin. All rights reserved. Contacts: <claygod@yandex.ru>

import (
	"bytes"
	"io"
	"runtime"
	"sync/atomic"
)

const batchRatio int = 10 // how many batches fit into the input channel

const (
	stateStop int64 = 0 << iota
	stateStart
)

/*
Batcher - performs write jobs in batches.
*/
type Batcher struct {
	indicator *Indicator
	work      io.Writer
	alarm     func(error)
	chInput   chan []byte
	chStop    chan struct{}
	batchSize int
	stopFlag  int64
}

/*
NewBatcher - create new batcher.
Arguments:
	- indicator - the closing of the channels signals the completed task
	- work  - function that records the formed batch
	- alarm - error handling function
	- chInput - input channel
	- chStop - channel for the correct stoppage of the worker
	- batchSize - batch size
*/
func NewBatcher(i *Indicator, work io.Writer, alarm func(error), chInput chan []byte, batchSize int) *Batcher {
	return &Batcher{
		indicator: i,
		work:      work,
		alarm:     alarm,
		chInput:   chInput,
		chStop:    make(chan struct{}, batchRatio*batchSize),
		batchSize: batchSize,
	}
}

/*
Start - run a worker
*/
func (b *Batcher) Start() {
	atomic.StoreInt64(&b.stopFlag, stateStart)
	b.worker()
}

/*
Stop - finish the job
*/
func (b *Batcher) Stop() {
	b.chStop <- struct{}{}
	for {
		if atomic.LoadInt64(&b.stopFlag) == stateStop {
			return
		}
		runtime.Gosched()
	}
}

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
		for i := 0; i < b.batchSize; i++ {
			select {
			case inData := <-b.chInput:
				buf.Write(inData)
				if _, err := buf.Write(inData); err != nil {
					b.alarm(err)
				}
			default:
				runtime.Gosched()
				break
			}
		}

		if _, err := b.work.Write(buf.Bytes()); err != nil {
			b.alarm(err)
		}

		select {
		case <-b.chStop:
			atomic.StoreInt64(&b.stopFlag, stateStop)
			return
		default:
		}

		b.indicator.SwitchChan()
		buf.Reset()
	}
}
