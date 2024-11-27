package errstore

// Error storage
// Tests
// Copyright © 2024 Eduard Sesigin. All rights reserved. Contacts: <claygod@yandex.ru>

import (
	"fmt"
	"testing"
)

const (
	emptyString = ""
)

var testErrFoo = fmt.Errorf("foo")
var testHaveWant = "Have `%s` want `%s`"
var testErrShort = "1 errors received"
var testErrFull = "1 errors received: [error 0] foo"

func TestErrStoreEmpty(t *testing.T) {
	es := NewErrStore()

	if es.Count() != 0 {
		t.Errorf("Have %d want 0", es.Count())
	}

	if es.Error() != emptyString {
		t.Errorf(testHaveWant, es.Error(), emptyString)
	}
}

func TestErrStoreCount(t *testing.T) {
	es := NewErrStore()
	es.Add(fmt.Errorf("foo"))
	es.Add(fmt.Errorf("bar"))

	if es.Count() != 2 {
		t.Errorf(testHaveWant, es.Count(), "2")
	}
}

func TestErrStoreModeDefault(t *testing.T) {
	es := NewErrStore()
	es.Add(testErrFoo)

	if es.Error() != testErrShort {
		t.Errorf(testHaveWant, es.Error(), testErrShort)
	}
}

func TestErrStoreShort(t *testing.T) {
	es := NewErrStore().SetShortMode()
	es.Add(testErrFoo)

	if es.Error() != testErrShort {
		t.Errorf(testHaveWant, es.Error(), testErrShort)
	}
}

func TestErrStoreFull(t *testing.T) {
	es := NewErrStore().SetFullMode()
	es.Add(testErrFoo)

	if es.Error() != testErrFull {
		t.Errorf(testHaveWant, es.Error(), testErrFull)
	}
}

func TestErrStoreAppend(t *testing.T) {
	es := NewErrStore()
	es2 := NewErrStore()
	es2.Add(testErrFoo)
	es.Append(es2)

	if es.Error() != testErrShort {
		t.Errorf(testHaveWant, es.Error(), testErrShort)
	}
}
