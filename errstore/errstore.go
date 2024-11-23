package errstore

import (
	"fmt"
	"sync"
)

const (
	ErrShort = false
	ErrFull  = true
)

type ErrStore struct {
	m        sync.Mutex
	errShort bool
	errs     []error
}

func NewErrStore() *ErrStore {
	return &ErrStore{
		m:        sync.Mutex{},
		errShort: ErrShort,
		errs:     make([]error, 0),
	}
}

func (e *ErrStore) SetShortMode() *ErrStore {
	e.errShort = ErrShort

	return e
}

func (e *ErrStore) SetFullMode() *ErrStore {
	e.errShort = ErrFull

	return e
}

func (e *ErrStore) Count() int {
	e.m.Lock()
	defer e.m.Unlock()

	return len(e.errs)
}

func (e *ErrStore) Error() string {
	e.m.Lock()
	defer e.m.Unlock()

	if e.errShort == ErrShort {
		return e.short()
	}

	return e.full()
}

func (e *ErrStore) Add(err error) {
	e.m.Lock()
	defer e.m.Unlock()

	e.errs = append(e.errs, err)
}

func (e *ErrStore) Append(e2 *ErrStore) {
	e.m.Lock()
	defer e.m.Unlock()

	e.errs = append(e.errs, e2.errs...)
}

func (e *ErrStore) short() string {
	var out string

	if len(e.errs) > 0 {
		out = fmt.Sprintf("%d errors received", len(e.errs))
	}

	return out
}

func (e *ErrStore) full() string {
	var out string

	if len(e.errs) > 0 {
		out = fmt.Sprintf("%d errors received:", len(e.errs))

		for i, err := range e.errs {
			out = fmt.Sprintf("%s [error %d] %s", out, i, err.Error())
		}
	}

	return out
}
