package batcher

// Batcher
// Tests
// Copyright © 2018 Eduard Sesigin. All rights reserved. Contacts: <claygod@yandex.ru>

import (
	"os"
	"testing"
	"time"
)

func TestBatcher(t *testing.T) {
	fileName := "./test.txt"
	wr := newMockWriter(fileName)
	chIn := make(chan []byte, 100)
	batchSize := 10
	btch := NewBatcher(wr, mockAlarmHandle, chIn, batchSize)
	btch.Start()
	for u := 0; u < 25; u++ {
		chIn <- []byte{97}
	}
	time.Sleep(200 * time.Millisecond)
	wr.Close()
	f, _ := os.Open(fileName)
	st, err := f.Stat()
	if err != nil {
		t.Error("Error `stat` file")
	}
	if st.Size() != 28 {
		t.Error("Want 28, have ", st.Size())
	}

	btch.Stop()
	os.Remove(fileName)
}

// --- Helpers for tests ---

type mockWriter struct {
	f *os.File
}

func newMockWriter(fileName string) *mockWriter {
	f, _ := os.Create(fileName)
	return &mockWriter{
		f: f,
	}
}
func (m *mockWriter) Write(in []byte) (int, error) {
	m.f.Write(in)
	m.f.Write([]byte("\n")) // to calculate the batch
	return len(in), nil
}
func (m *mockWriter) Close() {
	m.f.Close()
}

func mockAlarmHandle(err error) {
	panic(err)
}
