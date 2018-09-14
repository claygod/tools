package worker

// Worker
// Tests
// Copyright © 2018 Eduard Sesigin. All rights reserved. Contacts: <claygod@yandex.ru>

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

/*
TestExampleOneWorker - one worker lags behind the generator
*/
func TestExampleOneWorker(t *testing.T) {
	ch := make(chan int, 1000)
	var stopGen int64 = stateStarted
	e := newExample(ch)
	w := New().Stoper(nil).Worker(e.work1)
	go genInputStream(ch, &stopGen)

	w.Start()
	time.Sleep(1 * time.Second)
	atomic.StoreInt64(&stopGen, stateStopped)
	w.Stop()
}

/*
TestExampleTwoIdenticalWorkers - two identical workers NOT behind the generator
*/
func TestExampleTwoIdenticalWorkers(t *testing.T) {
	ch := make(chan int, 1000)
	var stopGen int64 = stateStarted
	e := newExample(ch)
	w := New().Stoper(nil).Worker(e.work1).Worker(e.work1)
	go genInputStream(ch, &stopGen)

	w.Start()
	time.Sleep(1 * time.Second)
	atomic.StoreInt64(&stopGen, stateStopped)
	w.Stop()
}

/*
TestExampleTwoDifferentWorkers - two different workers NOT behind the generator
*/
func TestExampleTwoDifferentWorkers(t *testing.T) {
	ch := make(chan int, 1000)
	var stopGen int64 = stateStarted
	e := newExample(ch)
	w := New().Stoper(nil).Worker(e.work1).Worker(e.work2)
	go genInputStream(ch, &stopGen)

	w.Start()
	time.Sleep(1 * time.Second)
	atomic.StoreInt64(&stopGen, stateStopped)
	w.Stop()
}

// --- staff ---

func genInputStream(ch chan int, stopGen *int64) {
	for i := 0; i < 1000; i++ {
		fmt.Println("- genegator send ", i)
		ch <- i
		time.Sleep(50 * time.Millisecond)
		if atomic.LoadInt64(stopGen) == stateStopped {
			return
		}
	}
}

type example struct {
	ch chan int
}

func newExample(ch chan int) *example {
	return &example{
		ch: ch,
	}
}

func (e *example) work1() {
	count := <-e.ch
	fmt.Println("- worker (1) processing number ", count)
	time.Sleep(100 * time.Millisecond)
}

func (e *example) work2() {
	count := <-e.ch
	fmt.Println("- worker (2) processing number ", count)
	time.Sleep(200 * time.Millisecond)
}
