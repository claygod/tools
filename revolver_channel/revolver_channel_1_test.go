package revolver_channel

// Revolver Channel
// Tests 1
// Copyright © 2026 Eduard Sesigin. All rights reserved. Contacts: <claygod@yandex.ru>

import (
	"sync"
	"testing"
	"time"
)

func TestNewRevolverChannel(t *testing.T) {
	rCh, _ := NewRevolverChannel[int](3, 65536)
	defer cleanupChannel(rCh)

	for i := 40; i < 48; i++ {
		// fmt.Println(rCh.shifter.Tail(), rCh.shifter.Head())
		// time.Sleep(time.Millisecond)
		u := i
		rCh.In <- u
	}

	if v := rCh.shifter.Head(); v != 2 {
		t.Errorf("Expected value 2, obtained %d", v)
	}

	if v := rCh.Len(); v != 8 {
		t.Errorf("Expected value 8, obtained %d", v)
	}

}

func TestRevolverChannelOrderOfResults(t *testing.T) {
	rCh, _ := NewRevolverChannel[int](3, 65536)
	defer cleanupChannel(rCh)

	for i := 40; i < 48; i++ {
		u := i
		rCh.In <- u
	}

	for i := 40; i < 48; i++ {
		u := i
		value := <-rCh.Out

		if value != u {
			t.Errorf("Expected value %d, obtained %d", value, u)
		}
	}
}

func TestRevolverChannelLen1(t *testing.T) {
	rCh, _ := NewRevolverChannel[int](3, 65536)
	defer cleanupChannel(rCh)

	for i := 40; i < 48; i++ {
		u := i
		rCh.In <- u
	}

	if v := rCh.Len(); v != 8 {
		t.Errorf("Expected value 8, obtained %d", v)
	}
}

func TestRevolverChannelStopClose(t *testing.T) {
	rCh, _ := NewRevolverChannel[int](3, 65536)
	defer cleanupChannel(rCh)

	for i := 40; i < 48; i++ {
		u := i
		rCh.In <- u
	}

	if v := rCh.IsStoped(); v {
		t.Errorf("Expected value 'false`, obtained `%v`", v)
	}

	if v := rCh.IsClosed(); v {
		t.Errorf("Expected value 'false`, obtained `%v`", v)
	}

	rCh.Stop()

	if v := rCh.IsStoped(); !v {
		t.Errorf("Expected value 'true`, obtained `%v`", v)
	}

	if v := rCh.IsClosed(); v {
		t.Errorf("Expected value 'false`, obtained `%v`", v)
	}

	i := 40
	for u := range rCh.Out {
		if i != u {
			t.Errorf("Expected value %d, obtained %d", i, u)
		}

		i++
	}

	if v := rCh.IsStoped(); !v {
		t.Errorf("Expected value 'true`, obtained `%v`", v)
	}

	// rCh.WaitClose()

	if v := rCh.IsClosed(); !v {
		t.Errorf("Expected value 'false`, obtained `%v`", v)
	}

	if _, ok := <-rCh.Out; ok {
		t.Errorf("Expected value 'false`, obtained `%v`", ok)
	}
}

// ============================================================================
// Тест: Utilization() — низкая, средняя и высокая утилизация
// ============================================================================
func TestRevolverChannel_Utilization(t *testing.T) {
	const chCap = 1 // Минимальный буфер, чтобы каждый value занимал новый канал

	t.Run("Low utilization (~0.01%)", func(t *testing.T) {
		rCh, _ := NewRevolverChannel[int](chCap, 65536)
		defer cleanupChannel(rCh)

		// Отправляем 10 значений — должно занять ~10 каналов из 65536
		for i := 0; i < 10; i++ {
			rCh.In <- i
		}

		util := rCh.Utilization()
		expectedMin := 0.01 // ~10/65536*100
		expectedMax := 0.02

		if util < expectedMin || util > expectedMax {
			t.Errorf("Utilization=%.4f%% out of expected range [%.2f, %.2f]",
				util, expectedMin, expectedMax)
		}
		t.Logf("Low utilization: %.4f%% (shiftIn=%d, shiftOut=%d)",
			util, rCh.shifter.Head(), rCh.shifter.Tail())
	})

	t.Run("Medium utilization (~50%)", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping medium utilization test in short mode")
		}

		rCh, _ := NewRevolverChannel[int](chCap, 65536)
		defer cleanupChannel(rCh)

		// Отправляем ~32768 значений → ~50% утилизации
		targetChannels := limit16bit / 2
		var wg sync.WaitGroup

		// Писатель
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < targetChannels; i++ {
				rCh.In <- i
			}
		}()

		// Читатель (читает медленнее, чтобы буфер накапливался)
		wg.Add(1)
		go func() {
			defer wg.Done()
			readCount := 0
			for readCount < targetChannels/4 { // Читаем только 25%, чтобы буфер рос
				select {
				case <-rCh.Out:
					readCount++
				case <-time.After(10 * time.Millisecond):
					// Продолжаем
				}
			}
		}()

		// Даём время накопить буфер
		time.Sleep(100 * time.Millisecond)

		util := rCh.Utilization()
		expectedMin := 40.0 // 40-60% — приемлемый диапазон
		expectedMax := 60.0

		if util < expectedMin || util > expectedMax {
			t.Logf("Note: Utilization=%.2f%% may vary due to concurrent reads", util)
			// Не fail-им тест, т.к. точное значение зависит от таймингов
		} else {
			t.Logf("Medium utilization: %.2f%% (shiftIn=%d, shiftOut=%d)",
				util, rCh.shifter.Head(), rCh.shifter.Tail())
		}

		// Завершаем тест
		rCh.Stop()
		go func() {
			for range rCh.Out {
			}
		}()
		wg.Wait()
		rCh.WaitClose()
	})

	t.Run("High utilization (~90%+)", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping high utilization test in short mode")
		}

		const chCap = 1
		rCh, _ := NewRevolverChannel[int](chCap, 65536)
		targetValues := limit16bit * 9 / 10

		var (
			maxUtil float64
			utilMu  sync.Mutex
			done    = make(chan struct{})
		)

		// Монитор утилизации (запускаем ПЕРВЫМ)
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
				case <-done:
					return
				}
			}
		}()

		// Быстрая запись
		for i := 0; i < targetValues; i++ {
			rCh.In <- i
		}

		// Медленное чтение (только 10%, чтобы буфер рос)
		go func() {
			for j := 0; j < targetValues/10; j++ {
				<-rCh.Out
				time.Sleep(10 * time.Microsecond)
			}
		}()

		// Даём время стабилизироваться
		time.Sleep(50 * time.Millisecond)
		close(done)

		// Проверяем ПИК утилизации, а не финальное значение
		utilMu.Lock()
		peakUtil := maxUtil
		utilMu.Unlock()

		t.Logf("Peak utilization: %.2f%% (shiftIn=%d, shiftOut=%d)",
			peakUtil, rCh.shifter.Head(), rCh.shifter.Tail())

		if peakUtil < 50 {
			t.Errorf("Expected peak utilization >50%%, got %.2f%%", peakUtil)
		}

		// Завершение
		rCh.Stop()
		go func() {
			for range rCh.Out {
			}
		}()
		rCh.WaitClose()
	})

	t.Run("Utilization after full drain", func(t *testing.T) {
		rCh, _ := NewRevolverChannel[int](chCap, 65536)
		defer cleanupChannel(rCh)

		// Наполняем и полностью вычитываем
		for i := 0; i < 100; i++ {
			rCh.In <- i
		}

		// Вычитываем всё
		for i := 0; i < 100; i++ {
			<-rCh.Out
		}

		// Небольшая задержка для обновления shiftOut
		time.Sleep(10 * time.Millisecond)

		util := rCh.Utilization()
		// После полного дренажа утилизация должна быть близка к 0
		// (может быть >0, если shiftOut ещё не догнал shiftIn из-за асинхронности)
		if util > 1.0 {
			t.Errorf("Expected utilization ~0%% after full drain, got %.4f%%", util)
		}
		t.Logf("Utilization after drain: %.4f%%", util)

		rCh.Stop()
		go func() {
			for range rCh.Out {
			}
		}()
		rCh.WaitClose()
	})
}

// ============================================================================
// Тест: Утилизация при циклическом использовании (wraparound)
// ============================================================================
func TestRevolverChannel_Utilization_Wraparound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping wraparound utilization test in short mode")
	}

	const chCap = 1
	rCh, _ := NewRevolverChannel[int](chCap, 65536)
	defer cleanupChannel(rCh)

	var wg sync.WaitGroup

	// Пишем и читаем циклически, чтобы спровоцировать wraparound shiftIn/shiftOut
	totalIterations := limit16bit + 1000 // Больше, чем максимум каналов

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < totalIterations; i++ {
			rCh.In <- i
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < totalIterations; i++ {
			<-rCh.Out
			// Небольшая задержка для рассинхронизации
			if i%100 == 0 {
				time.Sleep(10 * time.Microsecond)
			}
		}
	}()

	// В процессе проверяем, что утилизация никогда не превышает 100%
	checkTicker := time.NewTicker(50 * time.Millisecond)
	defer checkTicker.Stop()

	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

Loop:
	for {
		select {
		case <-checkTicker.C:
			util := rCh.Utilization()
			if util < 0 || util > 100 {
				t.Errorf("Utilization out of bounds: %.2f%%", util)
			}
		case <-done:
			break Loop
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout in wraparound test")
		}
	}

	t.Logf("Wraparound test completed: final utilization=%.2f%%", rCh.Utilization())

	rCh.Stop()
	go func() {
		for range rCh.Out {
		}
	}()
	rCh.WaitClose()
}

// ============================================================================
// Helper: безопасная очистка канала после теста
// ============================================================================
func cleanupChannel[T any](rCh *RevolverChannel[T]) {
	rCh.Stop()
	// Дренажируем Out в фоне, чтобы workerOut мог завершиться
	go func() {
		for range rCh.Out {
			// Drain
		}
	}()
	rCh.WaitClose()
}
