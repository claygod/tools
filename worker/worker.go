package worker

// Worker
// API
// Copyright © 2018 Eduard Sesigin. All rights reserved. Contacts: <claygod@yandex.ru>

import (
	"fmt"
	"runtime"
	"sync/atomic"
)

const (
	stateStopped int64 = iota
	stateStarted
)

/*
Worker - universal router for start-stop services
*/
type Worker struct {
	hasp        int64
	startFunc   func() error
	stopFunc    func() error
	workerFuncs []func()
}

/*
NewWorker - create new worker.
*/
func New() *Worker {
	return &Worker{
		hasp:        stateStopped,
		workerFuncs: make([]func(), 0, 1),
	}
}

/*
SetStartFunc -
*/
func (w *Worker) Starter(f func() error) *Worker {
	w.startFunc = f
	return w
}

/*
SetStopFunc -
*/
func (w *Worker) Stoper(f func() error) *Worker {
	w.stopFunc = f
	return w
}

/*
SetStopFunc -
*/
func (w *Worker) Worker(f func()) *Worker {
	w.workerFuncs = append(w.workerFuncs, f)
	return w
}

/*
Start -
*/
func (w *Worker) Start() error {
	if len(w.workerFuncs) == 0 {
		return fmt.Errorf("No functions for work (for start)")
	}
	state := atomic.LoadInt64(&w.hasp)
	switch {
	case state > stateStopped: // already started
		return nil
	case state < stateStopped: // already stops
		for {
			if atomic.LoadInt64(&w.hasp) == stateStopped {
				break
			}
			runtime.Gosched()
		}
		fallthrough
	case state == stateStopped: // stopped, you can start
		for _, f := range w.workerFuncs {
			if w.startFunc != nil {
				if err := w.startFunc(); err != nil {
					w.Stop()
					return err
				}
			}
			atomic.AddInt64(&w.hasp, 1)
			go w.worker(f)
		}
		return nil
	}
	return nil
}

/*
Stop -
*/
func (w *Worker) Stop() error {
	if len(w.workerFuncs) == 0 {
		return fmt.Errorf("No functions for work  (for stop)")
	}
	state := atomic.LoadInt64(&w.hasp)
	switch {
	case state == stateStopped:
		return nil
	case state > stateStopped: // already stops
		atomic.StoreInt64(&w.hasp, -state) // TODO: unsafe
		fallthrough
	case state < stateStopped:
		for {
			if atomic.LoadInt64(&w.hasp) == stateStopped {
				if w.stopFunc != nil {
					return w.stopFunc()
				}
				return nil
			}
			runtime.Gosched()
		}
	}
	return nil
}

func (w *Worker) worker(f func()) {
	for {
		f()
		if atomic.LoadInt64(&w.hasp) < stateStopped {
			atomic.AddInt64(&w.hasp, 1)
			return
		}
	}
}
