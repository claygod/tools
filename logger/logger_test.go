package logger

// Logger
// Tests
// Copyright © 2018 Eduard Sesigin. All rights reserved. Contacts: <claygod@yandex.ru>

import (
	"fmt"
	"log"
	"testing"
)

func TestLoggerError(t *testing.T) {
	lgr := New(&mockWriterTrue{})

	count, err := lgr.Error("testing message").Send()
	if err != nil {
		t.Error(err)
	}

	if count != 24 {
		t.Error("Invalid number of characters sent to the log: ", count)
	}
}

func TestLoggerWarning(t *testing.T) {
	lgr := New(&mockWriterTrue{})

	count, err := lgr.Warning("testing message").Send()
	if err != nil {
		t.Error(err)
	}

	if count != 26 {
		t.Error("Invalid number of characters sent to the log: ", count)
	}
}

func TestLoggerContext(t *testing.T) {
	lgr := New(&mockWriterTrue{}).Context("AAA", "aaa")

	count, err := lgr.Error("testing message").Send()
	if err != nil {
		t.Error(err)
	}

	if count != 34 {
		t.Error("Invalid number of characters sent to the log: ", count)
	}
}

func TestLoggerBranchingLen(t *testing.T) {
	lgr := New(&mockWriterTrue{})
	lgrA := lgr.Context("BranchA", "333")
	lgrB := lgr.Context("BranchB", "55555")
	countA, _ := lgrA.Send()
	countB, _ := lgrB.Send()

	if countA != 14 || countB != 16 {
		t.Error("Invalid number of characters sent to the log when branching ", countA, countB)
	}
}
func TestLoggerBranchingContext(t *testing.T) {
	lgr := New(&mockWriterTrue{}).Info("Hello world")
	lgrA := lgr.Context("BranchA", "333")
	lgrB := lgr.Context("BranchB", "55555")

	if lgrA.parent.context != lgrB.parent.context {
		t.Error("Error communicating with parent structure ", lgrA.parent, lgrB.parent)
	} else if lgrA.parent.context != "Hello world" {
		t.Error("Error communicating with parent structure ", lgrA.parent, lgrB.parent)
	}
}

func TestLoggerFakeWriter(t *testing.T) {
	_, err := New(&mockWriterFalse{}).Info("Hello world").Send()

	if err == nil {
		t.Error("There should have been an error")
	}
}

/*
mockWriterTrue - TRUE mock for the interface of the Writer
*/
type mockWriterTrue struct{}

func (m *mockWriterTrue) Write(b []byte) (int, error) {
	log.Print(string(b))

	return len(b), nil
}

/*
mockWriterFalse - FALSE mock for the interface of the Writer
*/
type mockWriterFalse struct{}

func (m *mockWriterFalse) Write(b []byte) (int, error) {
	return len(b), fmt.Errorf("Specially Generated Error")
}
