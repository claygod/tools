package splitwork

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/claygod/tools/errstore"
)

type Worker[T any, R any] struct {
	wFunc    func(T) (R, error)
	wCount   int
	errShort bool
}

func NewWorker[T any, R any](wFunc func(T) (R, error)) *Worker[T, R] {
	return &Worker[T, R]{
		wFunc:    wFunc,
		wCount:   runtime.NumCPU(),  // default
		errShort: errstore.ErrShort, // default
	}
}

func (w *Worker[T, R]) SetShortModeErr() *Worker[T, R] {
	w.errShort = errstore.ErrShort

	return w
}

func (w *Worker[T, R]) SetFullModeErr() *Worker[T, R] {
	w.errShort = errstore.ErrFull

	return w
}

func (w *Worker[T, R]) SetWorkersCount(count int) *Worker[T, R] {
	if count > 0 {
		w.wCount = count
	}

	return w
}

func (w *Worker[T, R]) Do(ctx context.Context, items []T) RespWrap[R] {
	errMode := w.errShort
	doneChan := ctx.Done()
	workChan := make(chan T)
	respChan := make(chan RespWrap[R])

	// worker
	wf := func(chIn chan T, chOut chan RespWrap[R]) {
		errs := errstore.NewErrStore(errMode)
		rList := make([]R, 0)

		for in := range chIn {
			fmt.Println("Обрабатываю ", in)
			resp, err := w.wFunc(in)
			if err != nil {
				errs.Add(err)
			} else {
				rList = append(rList, resp)
			}

			time.Sleep(2 * time.Second)
		}

		chOut <- RespWrap[R]{Data: rList, Err: errs}
	}

	// launch workers
	for i := 0; i < w.wCount; i++ {
		go wf(workChan, respChan)
	}

	// work cycle of sending on input array
workForList:

	for _, v := range items {
		select {
		case <-doneChan:
			fmt.Println("Сработал прерыватель! ", v)
			break workForList

		default:
			workChan <- v
		}
	}

	close(workChan)

	// collecting results of workers' work
	rData := make([]R, 0, len(items))
	rErr := errstore.NewErrStore(errMode)

	for i := 0; i < w.wCount; i++ {
		fmt.Println("Получение ", i)
		wResp := <-respChan

		rErr.Add(wResp.Err)
		rData = append(rData, wResp.Data...)
	}

	return RespWrap[R]{
		Data: rData,
		Err:  rErr,
	}
}

type RespWrap[R any] struct {
	Data []R
	Err  error
}
