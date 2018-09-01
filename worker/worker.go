package worker

// Worker
// Copyright © 2018 Eduard Sesigin. All rights reserved. Contacts: <claygod@yandex.ru>

import (
	"runtime"
	"sync/atomic"
)

const (
	stateStopped int64 = iota
	stateStarted
)

/*
NewWorker - create new worker.
*/
func NewWorker(startFunc func() error, stopFunc func() error, workerFuncs []func()) *Worker {
	return &Worker{
		hasp:        stateStopped,
		startFunc:   startFunc,
		stopFunc:    stopFunc,
		workerFuncs: workerFuncs,
	}
}

/*
SetStartFunc -
*/
func (w *Worker) SetStartFunction(f func() error) {
	w.startFunc = f
}

/*
SetStopFunc -
*/
func (w *Worker) SetStopFunction(f func() error) {
	w.stopFunc = f
}

/*
SetStopFunc -
*/
func (w *Worker) SetWorkerFunctions(fs []func()) {
	w.workerFuncs = fs
}

/*
Start -
*/
func (w *Worker) Start() error {
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
