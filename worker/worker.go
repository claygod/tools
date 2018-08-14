package worker

// Worker
// Copyright © 2018 Eduard Sesigin. All rights reserved. Contacts: <claygod@yandex.ru>

import (
	// "fmt"
	"runtime"
	"sync/atomic"
)

const (
	stateStopped int64 = iota
	stateStarted
)

/*
Worker -
*/
type Worker struct {
	hasp       int64
	startFunc  func() error
	stopFunc   func() error
	workerFunc func()
}

func NewWorker(startFunc func() error, stopFunc func() error, workerFunc func()) *Worker {
	return &Worker{
		hasp:       stateStopped,
		startFunc:  startFunc,
		stopFunc:   stopFunc,
		workerFunc: workerFunc,
	}
}

/*
Start -
*/
func (w *Worker) Start() error {
	state := atomic.LoadInt64(&w.hasp)
	switch {
	case state > stateStopped: // уже запущен

		return nil
	case state < stateStopped: // останавливается
		for {
			if atomic.LoadInt64(&w.hasp) == stateStopped {
				break
			}
			runtime.Gosched()
		}
		fallthrough
	case state == stateStopped: // стоит, можно запускать
		atomic.AddInt64(&w.hasp, 1)
		if err := w.startFunc(); err != nil {
			atomic.StoreInt64(&w.hasp, stateStopped)
			return err
		}
		return nil

	}
	return nil
}

/*
Stop -
*/
func (w *Worker) Stop() error {
	state := atomic.LoadInt64(&w.hasp)
	switch {
	case state == stateStopped:
		return nil
	case state > stateStopped: // уже останавливается
		atomic.StoreInt64(&w.hasp, -state) // TODO: не совсем безопасно реализовано
		fallthrough
	case state < stateStopped:
		for {
			if atomic.LoadInt64(&w.hasp) == stateStopped {
				return w.stopFunc()
			}
			runtime.Gosched()
		}
	}
	return nil
}

func (w *Worker) worker() {
	for {
		w.workerFunc()
		// --

		if atomic.LoadInt64(&w.hasp) < stateStopped {
			atomic.AddInt64(&w.hasp, 1)
			return
		}
	}
}
