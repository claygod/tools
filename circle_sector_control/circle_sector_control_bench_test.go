package circle_sector_control

// Circle sector control
// Tests 1
// Copyright © 2026 Eduard Sesigin. All rights reserved. Contacts: <claygod@yandex.ru>

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

// ==================== Базовые бенчмарки ====================

func BenchmarkHeadForward_SingleThread(b *testing.B) {
	c := NewCircleSectorControl(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.HeadForward()
	}
}

func BenchmarkTailForward_SingleThread(b *testing.B) {
	c := NewCircleSectorControl(1000)
	// Предварительно двигаем head, чтобы tail мог двигаться
	for i := 0; i < 100; i++ {
		_, _ = c.HeadForward()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.TailForward()
	}
}

func BenchmarkUtilization_SingleThread(b *testing.B) {
	c := NewCircleSectorControl(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.Utilization()
	}
}

// ==================== Конкурентные бенчмарки ====================

// BenchmarkHeadForward_Concurrent — несколько горутин двигают head
func BenchmarkHeadForward_Concurrent(b *testing.B) {
	c := NewCircleSectorControl(1000)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = c.HeadForward()
		}
	})
}

// BenchmarkTailForward_Concurrent — несколько горутин двигают tail
func BenchmarkTailForward_Concurrent(b *testing.B) {
	c := NewCircleSectorControl(1000)
	// Предварительно создаём "пространство" для tail
	for i := 0; i < 500; i++ {
		_, _ = c.HeadForward()
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = c.TailForward()
		}
	})
}

// BenchmarkHeadTail_Concurrent — head и tail двигаются параллельно
// Используем атомарный счётчик для чередования операций
func BenchmarkHeadTail_Concurrent(b *testing.B) {
	c := NewCircleSectorControl(1000)
	var counter int64

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Чётные итерации — head, нечётные — tail
			if atomic.AddInt64(&counter, 1)%2 == 0 {
				_, _ = c.HeadForward()
			} else {
				_, _ = c.TailForward()
			}
		}
	})
}

// BenchmarkHeadTail_Concurrent_Separate — два отдельных пула горутин
// Более чистый тест: одни горутины только head, другие только tail
func BenchmarkHeadTail_Concurrent_Separate(b *testing.B) {
	c := NewCircleSectorControl(1000)

	b.ResetTimer()

	// Запускаем два параллельных бенчмарка
	b.Run("head_workers", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = c.HeadForward()
			}
		})
	})

	b.Run("tail_workers", func(b *testing.B) {
		// Предварительно освобождаем место для tail
		for i := 0; i < 500; i++ {
			_, _ = c.HeadForward()
		}
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = c.TailForward()
			}
		})
	})
}

// BenchmarkHeadTail_Concurrent_WithUtilization — с периодическим чтением утилизации
func BenchmarkHeadTail_Concurrent_WithUtilization(b *testing.B) {
	c := NewCircleSectorControl(1000)
	var counter int64

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if atomic.AddInt64(&counter, 1)%2 == 0 {
				_, _ = c.HeadForward()
			} else {
				_, _ = c.TailForward()
			}
			// Каждый 100-й вызов — чтение утилизации
			if atomic.LoadInt64(&counter)%100 == 0 {
				_ = c.Utilization()
			}
		}
	})
}

// ==================== Бенчмарки с разным размером cap ====================

func BenchmarkHeadForward_CapSmall(b *testing.B) {
	c := NewCircleSectorControl(10)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = c.HeadForward()
		}
	})
}

func BenchmarkHeadForward_CapLarge(b *testing.B) {
	c := NewCircleSectorControl(65535)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = c.HeadForward()
		}
	})
}

// BenchmarkHeadTail_Concurrent_CapVariations — матрица бенчмарков по размеру cap
func BenchmarkHeadTail_Concurrent_CapVariations(b *testing.B) {
	caps := []int64{10, 100, 1000, 10000}

	for _, cap := range caps {
		b.Run(fmt.Sprintf("cap_%d", cap), func(b *testing.B) {
			c := NewCircleSectorControl(cap)
			var counter int64

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					if atomic.AddInt64(&counter, 1)%2 == 0 {
						_, _ = c.HeadForward()
					} else {
						_, _ = c.TailForward()
					}
				}
			})
		})
	}
}

// ==================== Бенчмарки с блокировками (contention) ====================

func BenchmarkHeadForward_WithContention(b *testing.B) {
	c := NewCircleSectorControl(10) // Малый cap = высокая конкуренция

	// Заполняем буфер на 80%
	for i := 0; i < 8; i++ {
		_, _ = c.HeadForward()
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = c.HeadForward()
		}
	})
}

func BenchmarkTailForward_WithContention(b *testing.B) {
	c := NewCircleSectorControl(10)

	// Создаём состояние близкое к минимальному
	for i := 0; i < 9; i++ {
		_, _ = c.HeadForward()
	}
	for i := 0; i < 8; i++ {
		_, _ = c.TailForward()
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = c.TailForward()
		}
	})
}

// ==================== Бенчмарки атомарных операций (база для сравнения) ====================

func BenchmarkAtomic_LoadInt64(b *testing.B) {
	var val int64

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = atomic.LoadInt64(&val)
	}
}

func BenchmarkAtomic_CAS_Int64(b *testing.B) {
	var val int64

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for {
			cur := atomic.LoadInt64(&val)
			if atomic.CompareAndSwapInt64(&val, cur, cur+1) {
				break
			}
		}
	}
}

func BenchmarkCircleSector_FullCycle(b *testing.B) {
	c := NewCircleSectorControl(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Заполняем
		for j := 0; j < 99; j++ {
			_, _ = c.HeadForward()
		}
		// Опустошаем
		for j := 0; j < 99; j++ {
			_, _ = c.TailForward()
		}
	}
}

// ==================== Бенчмарки для сценариев RevolverChannel ====================

func BenchmarkRevolverChannel_Typical(b *testing.B) {
	c := NewCircleSectorControl(100)
	var wg sync.WaitGroup

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(2)

		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_, _ = c.HeadForward()
			}
		}()

		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_, _ = c.TailForward()
			}
		}()

		wg.Wait()
	}
}

func BenchmarkRevolverChannel_HighLoad(b *testing.B) {
	c := NewCircleSectorControl(1000)
	var counter int64

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if atomic.AddInt64(&counter, 1)%2 == 0 {
				_, _ = c.HeadForward()
			} else {
				_, _ = c.TailForward()
			}
		}
	})
}

// ==================== Вспомогательные бенчмарки ====================

func BenchmarkUtilization_DifferentStates(b *testing.B) {
	states := []struct {
		name string
		head int64
		tail int64
	}{
		{"min", 0, 0},
		{"half", 50, 0},
		{"full", 99, 0},
		{"wrap", 10, 90},
	}

	for _, st := range states {
		b.Run(st.name, func(b *testing.B) {
			c := &CircleSectorControl{cap: 100, head: st.head, tail: st.tail}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = c.Utilization()
			}
		})
	}
}

// BenchmarkHeadTail_ContentionProfile — профилирование частоты отказов
func BenchmarkHeadTail_ContentionProfile(b *testing.B) {
	c := NewCircleSectorControl(10) // Малый cap = высокая конкуренция

	var success, fail int64

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var counter int64
		for pb.Next() {
			var ok bool
			if atomic.AddInt64(&counter, 1)%2 == 0 {
				_, ok = c.HeadForward()
			} else {
				_, ok = c.TailForward()
			}
			if ok {
				atomic.AddInt64(&success, 1)
			} else {
				atomic.AddInt64(&fail, 1)
			}
		}
	})

	b.Logf("success=%d, fail=%d, fail_rate=%.2f%%",
		success, fail, float64(fail)*100/float64(success+fail))
}
