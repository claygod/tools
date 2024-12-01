package splitwork

// Split work
// Tests
// Copyright © 2024 Eduard Sesigin. All rights reserved. Contacts: <claygod@yandex.ru>

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"
)

const (
	testDur100 = 100 * time.Millisecond
)

func TestSplitWorkEasy(t *testing.T) {
	w := NewWorker(testWorkInc).SetWorkersCount(3)

	resp := w.Do(context.Background(), []int{1, 2, 3, 4, 5, 6})

	if w.wCount != 3 {
		t.Errorf("Have %d want 3", w.wCount)
	}

	if len(resp.Data) != 6 {
		t.Errorf("Data list: have %d want 6", len(resp.Data))
	}

	if resp.Err.Count() != 0 {
		t.Errorf("Data list: have %d want 0", resp.Err.Count())
	}
}

func TestSplitWorkTime(t *testing.T) {
	w := NewWorker(testWork1Sec).SetWorkersCount(2)
	curBegin := time.Now()

	w.Do(context.Background(), []int{1, 2, 3, 4, 5, 6})

	curFinish := time.Now()

	if curBegin.Add(time.Second).Before(curFinish) {
		t.Errorf("Parallelism not found. Begin: %v Finish: %v", curBegin, curFinish)
	}
}

func TestSplitWorkContext(t *testing.T) {
	w := NewWorker(testWork1Sec).SetWorkersCount(2)

	ctx, _ := context.WithTimeout(context.Background(), testDur100)
	resp := w.Do(ctx, []int{1, 2, 3, 4, 5, 6})

	if len(resp.Data) != 3 {
		t.Errorf("Data list: have %d want 3", len(resp.Data))
	}

	if resp.Err.Count() != 0 {
		t.Errorf("Data list: have %d want 0", resp.Err.Count())
	}
}

func TestSplitWorkWithError(t *testing.T) {
	w := NewWorker(testWorkErr).SetWorkersCount(3)

	resp := w.Do(context.Background(), []int{1, 2, 3, 4, 5, 6})

	if len(resp.Data) != 0 {
		t.Errorf("Data list: have %d want 0", len(resp.Data))
	}

	if resp.Err.Count() != 6 {
		t.Errorf("Data list: have %d want 6", resp.Err.Count())
	}
}

func testWorkInc(in int) (int, error) {
	return in + 1, nil
}

func testWorkErr(in int) (int, error) {
	return 0, errors.New("Error " + strconv.Itoa(in))
}

func testWork1Sec(in int) (int, error) {
	time.Sleep(testDur100 * 3)

	return in, nil
}
