package revolver_channel

// Revolver Channel
// Benchmark
// Copyright © 2026 Eduard Sesigin. All rights reserved. Contacts: claygod@yandex.ru

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const (
	defaultChCount = 65536 // Аналог 16-bit (2^16)
	lim16bit       = 65536
)

// ============================================================================
// Вспомогательные функции
// ============================================================================

// cleanupChannel безопасно завершает канал после бенчмарка
func cleanupBench[T any](b *testing.B, rCh *RevolverChannel[T]) {
	b.Helper()
	rCh.Stop()
	// Дренажируем Out в фоне, чтобы workerOut мог завершиться
	go func() {
		for range rCh.Out {
			// Drain
		}
	}()
	rCh.WaitClose()
}

// runBenchmark — общая логика для сингл-поточных бенчмарков
func runBenchmark[T any](b *testing.B, chCap, chCount int, payload T) {
	b.Helper()
	rCh, err := NewRevolverChannel[T](chCap, chCount)
	if err != nil {
		b.Fatalf("Failed to create channel: %v", err)
	}
	defer cleanupBench(b, rCh)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rCh.In <- payload
		<-rCh.Out
	}

	b.StopTimer()
}

// ============================================================================
// Бенчмарк: Throughput (пропускная способность)
// ============================================================================

func BenchmarkRevolverChannel_Throughput(b *testing.B) {
	const (
		chCap   = 100
		chCount = 65536
	)
	rCh, err := NewRevolverChannel[int](chCap, chCount)
	if err != nil {
		b.Fatalf("Failed to create channel: %v", err)
	}
	defer cleanupBench(b, rCh)

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
// Сингл-поточные бенчмарки (1 writer / 1 reader)
// ============================================================================

func BenchmarkRevolverChannel_Single_Int_Cap1(b *testing.B) {
	runBenchmark(b, 1, defaultChCount, 42)
}

func BenchmarkRevolverChannel_Single_Int_Cap10(b *testing.B) {
	runBenchmark(b, 10, defaultChCount, 42)
}

func BenchmarkRevolverChannel_Single_Int_Cap100(b *testing.B) {
	runBenchmark(b, 100, defaultChCount, 42)
}

func BenchmarkRevolverChannel_Single_String_Cap10(b *testing.B) {
	runBenchmark(b, 10, defaultChCount, "hello_world_payload")
}

func BenchmarkRevolverChannel_Single_Struct_Cap10(b *testing.B) {
	type Payload struct {
		ID    int64
		Data  [64]byte
		Flags uint32
	}
	runBenchmark(b, 10, defaultChCount, Payload{ID: 123, Flags: 0xFF})
}

// ============================================================================
// Параллельные бенчмарки с b.RunParallel
// ============================================================================

func runParallelBenchmarkFixed[T any](b *testing.B, chCap, chCount int, payload T) {
	b.Helper()
	rCh, err := NewRevolverChannel[T](chCap, chCount)
	if err != nil {
		b.Fatalf("Failed to create channel: %v", err)
	}
	defer cleanupBench(b, rCh)

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
func BenchmarkRevolverChannel_Parallel_8x8_Int_Cap10(b *testing.B) {
	runParallelBenchmarkFixed(b, 10, defaultChCount, 42)
}

func BenchmarkRevolverChannel_Parallel_8x8_Int_Cap100(b *testing.B) {
	runParallelBenchmarkFixed(b, 100, defaultChCount, 42)
}

// --- 32 writers / 32 readers эквивалент ---
func BenchmarkRevolverChannel_Parallel_32x32_Int_Cap10(b *testing.B) {
	runParallelBenchmarkFixed(b, 10, defaultChCount, 42)
}

func BenchmarkRevolverChannel_Parallel_32x32_Int_Cap100(b *testing.B) {
	runParallelBenchmarkFixed(b, 100, defaultChCount, 42)
}

// --- 8 writers / 32 readers (read-heavy) ---
func BenchmarkRevolverChannel_Parallel_8x32_Int_Cap10(b *testing.B) {
	runParallelBenchmarkFixed(b, 10, defaultChCount, 42)
}

func BenchmarkRevolverChannel_Parallel_8x32_Int_Cap100(b *testing.B) {
	runParallelBenchmarkFixed(b, 100, defaultChCount, 42)
}

// --- 32 writers / 8 readers (write-heavy) ---
func BenchmarkRevolverChannel_Parallel_32x8_Int_Cap10(b *testing.B) {
	runParallelBenchmarkFixed(b, 10, defaultChCount, 42)
}

func BenchmarkRevolverChannel_Parallel_32x8_Int_Cap100(b *testing.B) {
	runParallelBenchmarkFixed(b, 100, defaultChCount, 42)
}

// ============================================================================
// Бенчмарки с b.RunParallel (настоящая параллельность Go)
// ============================================================================

func runNativeParallelBenchmark[T any](b *testing.B, chCap, chCount int, payload T) {
	b.Helper()
	rCh, err := NewRevolverChannel[T](chCap, chCount)
	if err != nil {
		b.Fatalf("Failed to create channel: %v", err)
	}
	defer cleanupBench(b, rCh)

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

func BenchmarkRevolverChannel_NativeParallel_Int_Cap10(b *testing.B) {
	runNativeParallelBenchmark(b, 10, defaultChCount, 42)
}

func BenchmarkRevolverChannel_NativeParallel_Int_Cap100(b *testing.B) {
	runNativeParallelBenchmark(b, 100, defaultChCount, 42)
}

func BenchmarkRevolverChannel_NativeParallel_String_Cap10(b *testing.B) {
	runNativeParallelBenchmark(b, 10, defaultChCount, "benchmark_payload_string")
}

// ============================================================================
// Бенчмарк: измерение утилизации под нагрузкой
// ============================================================================

func BenchmarkRevolverChannel_UtilizationUnderLoad(b *testing.B) {
	const (
		chCap   = 10
		chCount = 65536
	)
	rCh, err := NewRevolverChannel[int](chCap, chCount)
	if err != nil {
		b.Fatalf("Failed to create channel: %v", err)
	}
	defer cleanupBench(b, rCh)

	var (
		maxUtil float64
		utilMu  sync.Mutex
		stopMon = make(chan struct{})
	)

	// Монитор утилизации
	go func() {
		ticker := time.NewTicker(100 * time.Microsecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				util := rCh.Utilization()
				utilMu.Lock()
				if util > maxUtil {
					maxUtil = util
				}
				utilMu.Unlock()
			case <-stopMon:
				return
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
	utilMu.Lock()
	b.ReportMetric(maxUtil, "max_utilization_percent")
	utilMu.Unlock()
}

// ============================================================================
// Бенчмарк: сравнение с обычным буферизованным каналом (baseline)
// ============================================================================

func BenchmarkNativeChannel_Single_Int_Cap10(b *testing.B) {
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

func BenchmarkNativeChannel_Parallel_8x8_Int_Cap10(b *testing.B) {
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

// ============================================================================
// Бенчмарк: малое количество каналов (для сравнения)
// ============================================================================

func BenchmarkRevolverChannel_Small_ChCount_100(b *testing.B) {
	runBenchmark(b, 10, 100, 42)
}

func BenchmarkRevolverChannel_Small_ChCount_1000(b *testing.B) {
	runBenchmark(b, 10, 1000, 42)
}

// ============================================================================
// Бенчмарк: утилизация при различной нагрузке
// ============================================================================

func BenchmarkRevolverChannel_Utilization_Low(b *testing.B) {
	const (
		chCap   = 1
		chCount = 65536
	)
	rCh, err := NewRevolverChannel[int](chCap, chCount)
	if err != nil {
		b.Fatalf("Failed to create channel: %v", err)
	}
	defer cleanupBench(b, rCh)

	b.ResetTimer()

	// Отправляем мало значений относительно chCount
	for i := 0; i < b.N/100; i++ {
		rCh.In <- i
		<-rCh.Out
	}

	b.ReportMetric(rCh.Utilization(), "avg_utilization_percent")
}

func BenchmarkRevolverChannel_Utilization_High(b *testing.B) {
	const (
		chCap   = 1
		chCount = 100 // Мало каналов → высокая утилизация
	)
	rCh, err := NewRevolverChannel[int](chCap, chCount)
	if err != nil {
		b.Fatalf("Failed to create channel: %v", err)
	}
	defer cleanupBench(b, rCh)

	var (
		maxUtil float64
		utilMu  sync.Mutex
		stopMon = make(chan struct{})
	)

	// Монитор утилизации
	go func() {
		ticker := time.NewTicker(50 * time.Microsecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				util := rCh.Utilization()
				utilMu.Lock()
				if util > maxUtil {
					maxUtil = util
				}
				utilMu.Unlock()
			case <-stopMon:
				return
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

	utilMu.Lock()
	b.ReportMetric(maxUtil, "max_utilization_percent")
	utilMu.Unlock()
}
