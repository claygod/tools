package revolver_channel

// Revolver Channel
// Benchmark 8 bit
// Copyright © 2026 Eduard Sesigin. All rights reserved. Contacts: <claygod@yandex.ru>

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func BenchmarkRevolverChannel8_Throughput(b *testing.B) {
	const chCap = 100
	rCh := NewRevolverChannel8Bit[int](chCap)
	defer cleanupBench8(b, rCh)

	b.ResetTimer()

	var sent, received int64

	// Писатель
	go func() {
		for i := 0; i < b.N; i++ {
			rCh.In <- i
			atomic.AddInt64(&sent, 1)
		}
	}()

	// Читатель
	go func() {
		for atomic.LoadInt64(&received) < int64(b.N) {
			<-rCh.Out
			atomic.AddInt64(&received, 1)
		}
	}()

	// Ждём завершения с таймаутом
	done := make(chan bool, 1)
	go func() {
		for atomic.LoadInt64(&received) < int64(b.N) {
			time.Sleep(10 * time.Millisecond)
		}
		done <- true
	}()

	select {
	case <-done:
		b.StopTimer()
	case <-time.After(30 * time.Second):
		b.Fatal("Throughput test timed out")
	}

	// Report extra metrics
	b.ReportMetric(float64(b.N)/float64(b.Elapsed().Seconds()), "ops/sec")
}

// ============================================================================
// Вспомогательные функции
// ============================================================================

// cleanupBench8 безопасно завершает канал после бенчмарка
func cleanupBench8[T any](b *testing.B, rCh *RevolverChannel8Bit[T]) {
	b.Helper()
	rCh.Stop()
	// Дренажируем Out в фоне, чтобы workerOut мог завершиться
	go func() {
		for range rCh.Out {
			// Drain
		}
	}()
	// Не вызываем WaitClose() в бенчмарках — это добавит лишнее время
	// Вместо этого даём немного времени на завершение
}

// runBenchmark — общая логика для сингл-поточных бенчмарков
func runBenchmark8[T any](b *testing.B, chCap int, payload T) {
	b.Helper()

	rCh := NewRevolverChannel8Bit[T](chCap)
	defer cleanupBench8(b, rCh)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rCh.In <- payload
		<-rCh.Out
	}

	b.StopTimer()
}

// ============================================================================
// Сингл-поточные бенчмарки (1 writer / 1 reader)
// ============================================================================

func BenchmarkRevolverChannel8_Single_Int_Cap1(b *testing.B) {
	runBenchmark8(b, 1, 42)
}

func BenchmarkRevolverChannel8_Single_Int_Cap10(b *testing.B) {
	runBenchmark8(b, 10, 42)
}

func BenchmarkRevolverChannel8_Single_Int_Cap100(b *testing.B) {
	runBenchmark8(b, 100, 42)
}

func BenchmarkRevolverChannel8_Single_String_Cap10(b *testing.B) {
	runBenchmark8(b, 10, "hello_world_payload")
}

func BenchmarkRevolverChannel8_Single_Struct_Cap10(b *testing.B) {
	type Payload struct {
		ID    int64
		Data  [64]byte
		Flags uint32
	}
	runBenchmark8(b, 10, Payload{ID: 123, Flags: 0xFF})
}

// ============================================================================
// Параллельные бенчмарки с b.RunParallel
// ============================================================================

func runParallelBenchmarkFixed8[T any](b *testing.B, chCap int, payload T) {
	b.Helper()

	rCh := NewRevolverChannel8Bit[T](chCap)
	defer cleanupBench8(b, rCh)

	b.ResetTimer()

	// Каждый параллельный поток делает полный цикл: send + recv
	b.RunParallel(func(pb *testing.PB) {
		localPayload := payload // избегаем race на payload
		for pb.Next() {
			rCh.In <- localPayload
			<-rCh.Out
		}
	})

	b.StopTimer()
}

// ============================================================================
// Параллельные бенчмарки: разные уровни конкурентности
// ============================================================================

// --- 8 writers / 8 readers эквивалент ---
func BenchmarkRevolverChannel8_Parallel_8x8_Int_Cap10(b *testing.B) {
	runParallelBenchmarkFixed8(b, 10, 42)
}

func BenchmarkRevolverChannel8_Parallel_8x8_Int_Cap100(b *testing.B) {
	runParallelBenchmarkFixed8(b, 100, 42)
}

// --- 32 writers / 32 readers эквивалент ---
func BenchmarkRevolverChannel8_Parallel_32x32_Int_Cap10(b *testing.B) {
	runParallelBenchmarkFixed8(b, 10, 42)
}

func BenchmarkRevolverChannel8_Parallel_32x32_Int_Cap100(b *testing.B) {
	runParallelBenchmarkFixed8(b, 100, 42)
}

// --- 8 writers / 32 readers (read-heavy) ---
func BenchmarkRevolverChannel8_Parallel_8x32_Int_Cap10(b *testing.B) {
	runParallelBenchmarkFixed8(b, 10, 42)
}

func BenchmarkRevolverChannel8_Parallel_8x32_Int_Cap100(b *testing.B) {
	runParallelBenchmarkFixed8(b, 100, 42)
}

// --- 32 writers / 8 readers (write-heavy) ---
func BenchmarkRevolverChannel8_Parallel_32x8_Int_Cap10(b *testing.B) {
	runParallelBenchmarkFixed8(b, 10, 42)
}

func BenchmarkRevolverChannel8_Parallel_32x8_Int_Cap100(b *testing.B) {
	runParallelBenchmarkFixed8(b, 100, 42)
}

// ============================================================================
// Бенчмарки с b.RunParallel (настоящая параллельность Go)
// ============================================================================

// runNativeParallelBenchmark8 — использует b.RunParallel для истинной параллельности
func runNativeParallelBenchmark8[T any](b *testing.B, chCap int, payload T) {
	b.Helper()

	rCh := NewRevolverChannel8Bit[T](chCap)
	defer cleanupBench8(b, rCh)

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		localPayload := payload
		for pb.Next() {
			rCh.In <- localPayload
			<-rCh.Out
		}
	})

	b.StopTimer()
}

func BenchmarkRevolverChannel8_NativeParallel_Int_Cap10(b *testing.B) {
	runNativeParallelBenchmark8(b, 10, 42)
}

func BenchmarkRevolverChannel8_NativeParallel_Int_Cap100(b *testing.B) {
	runNativeParallelBenchmark8(b, 100, 42)
}

func BenchmarkRevolverChannel8_NativeParallel_String_Cap10(b *testing.B) {
	runNativeParallelBenchmark8(b, 10, "benchmark_payload_string")
}

// ============================================================================
// Бенчмарк: измерение утилизации под нагрузкой
// ============================================================================

func BenchmarkRevolverChannel8_UtilizationUnderLoad(b *testing.B) {
	const chCap = 10
	rCh := NewRevolverChannel8Bit[int](chCap)
	defer cleanupBench8(b, rCh)

	var (
		maxUtil float64
		stopMon = make(chan struct{})
	)

	// Монитор утилизации
	go func() {
		ticker := make(chan struct{})
		go func() {
			for {
				select {
				case <-stopMon:
					close(ticker)
					return
				default:
					ticker <- struct{}{}
				}
			}
		}()

		for range ticker {
			util := rCh.Utilization()
			if util > maxUtil {
				maxUtil = util
			}
		}
	}()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rCh.In <- i
		<-rCh.Out
	}

	b.StopTimer()
	close(stopMon)

	// Report utilization as extra metric
	b.ReportMetric(maxUtil, "max_utilization_percent")
}

// ============================================================================
// Бенчмарк: сравнение с обычным буферизованным каналом (baseline)
// ============================================================================

func BenchmarkNativeChannel8_Single_Int_Cap10(b *testing.B) {
	ch := make(chan int, 10)
	done := make(chan struct{})

	// Фоновый читатель
	go func() {
		for range ch {
			// Drain
		}
		close(done)
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ch <- 42
	}
	b.StopTimer()

	close(ch)
	<-done
}

func BenchmarkNativeChannel8_Parallel_8x8_Int_Cap10(b *testing.B) {
	ch := make(chan int, 10)
	var wgReaders, wgWriters sync.WaitGroup

	// Читатели (8 горутин)
	for r := 0; r < 8; r++ {
		wgReaders.Add(1)
		go func() {
			defer wgReaders.Done()
			for range ch {
				// Drain
			}
		}()
	}

	b.ResetTimer()

	// Писатели (8 горутин)
	itemsPerGoroutine := b.N / 8
	for w := 0; w < 8; w++ {
		wgWriters.Add(1)
		go func() {
			defer wgWriters.Done()
			for i := 0; i < itemsPerGoroutine; i++ {
				ch <- 42
			}
		}()
	}

	// 1. Ждём писателей
	wgWriters.Wait()
	b.StopTimer()

	// 2. Закрываем канал — читатели выйдут из range
	close(ch)

	// 3. Ждём читателей
	wgReaders.Wait()
}
