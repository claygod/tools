package chbqueue

// Chanenel-based queue
// Tests
// Copyright © 2026 Eduard Sesigin. All rights reserved. Contacts: <claygod@yandex.ru>

import (
	"fmt"
	"testing"
	"time"
)

var forTestPass = "12345"

func TestNewKeybox(t *testing.T) {
	rCh := NewRevolverChannel16Bit(3)
	rCh.Start()

	for i := 40; i < 48; i++ {
		fmt.Println("ShIn|ShOut [0] ", rCh.shiftIn, rCh.shiftOut)
		time.Sleep(100 * time.Millisecond)
		u := i
		fmt.Println("Len|Cap [0] Before ", len(rCh.list[0]), cap(rCh.list[0]))
		fmt.Println("u=", u)
		time.Sleep(100 * time.Millisecond)
		rCh.In <- u
		time.Sleep(100 * time.Millisecond)
		fmt.Println("Len|Cap [0] After ", len(rCh.list[0]), cap(rCh.list[0]))
	}

	for v := range rCh.list[0] {
		fmt.Println("from ", v)
	}

	if v := rCh.shiftIn; v != 2 {

		t.Errorf("Expected value 2, obtained %d", v)
	}

}
