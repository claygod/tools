package batcher

// Batcher
// Batcher tests
// Copyright © 2018 Eduard Sesigin. All rights reserved. Contacts: <claygod@yandex.ru>

import (
	"os"
	"runtime/pprof"
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
	// os.Remove(fileName)
}

func BenchmarkClient(b *testing.B) { // go tool pprof -web ./batcher.test ./cpu.txt
	b.StopTimer()
	clt, err := Open("./tmp.txt", 2000)
	if err != nil {
		b.Error("Error `stat` file")
	}
	//defer
	dummy := forTestGetDummy(100)

	u := 0
	b.SetParallelism(256)
	f, err := os.Create("cpu.txt")
	if err != nil {
		b.Error("could not create CPU profile: ", err)
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		b.Error("could not start CPU profile: ", err)
	}
	b.StartTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			clt.Write(dummy)
			u++
		}
	})
	pprof.StopCPUProfile()
	clt.Close()
	// os.Remove(fileName)
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

func forTestGetDummy(count int) []byte {
	dummy := make([]byte, count)
	for i := 0; i < count; i++ {
		dummy[i] = 105
	}
	return dummy
}
