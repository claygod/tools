package chan_agent

// Channel Agent Tests
// Copyright © 2017-2018 Eduard Sesigin. All rights reserved. Contacts: <claygod@yandex.ru>

import (
	"runtime"
	"testing"
	"unsafe"
)

func TestChanAgentSendWitchPriority(t *testing.T) {
	ch := make(chan int64, 7)
	ag := runtime.NewChanAgent(ch)

	ch <- 700

	var item int64 = 200
	ag.Send(unsafe.Pointer(&item), true, false)

	if out := <-ch; out != 200 {
		t.Error("Want 200, have:", out)
	} else if out := <-ch; out != 700 {
		t.Error("Want 700, have:", out)
	}
}

func TestChanAgentSendWitchClean(t *testing.T) {
	ch := make(chan int64, 7)
	ag := runtime.NewChanAgent(ch)

	ch <- 700

	var item int64 = 200
	ag.Send(unsafe.Pointer(&item), true, true)

	if value := len(ch); value != 1 {
		t.Error("Want 1, have:", value)
	} else if out := <-ch; out != 200 {
		t.Error("Want 200, have:", out)
	}
}

func TestChanAgentClean(t *testing.T) {
	ch := make(chan int64, 7)
	ag := runtime.NewChanAgent(ch)

	ch <- 700
	ag.Clean()

	if value := len(ch); value != 0 {
		t.Error("Want 1, have:", value)
	}
}
