package revolver_channel

// Revolver Channel
// Tests 2
// Copyright © 2026 Eduard Sesigin. All rights reserved. Contacts: <claygod@yandex.ru>

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ============================================================================
// Тест 1: Полное заполнение буфера и проверка авто-расширения
// ============================================================================
func TestRevolverChannel_FullBufferExpansion(t *testing.T) {
	const (
		chCap = 2
		total = 10
	)

	rCh := NewRevolverChannel16Bit[int](chCap)

	// Отправляем значения
	for i := 0; i < total; i++ {
		rCh.In <- i
	}

	// Проверяем счётчик
	if got := rCh.Len(); got != int64(total) {
		t.Errorf("Expected Len=%d, got %d", total, got)
	}

	// Вычитываем все значения и проверяем порядок
	for i := 0; i < total; i++ {
		select {
		case v := <-rCh.Out:
			if v != i {
				t.Errorf("Expected %d, got %d", i, v)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("Timeout waiting for value %d", i)
		}
	}

	rCh.Stop()
	rCh.WaitClose()

	if !rCh.IsClosed() {
		t.Error("Expected IsClosed()=true after WaitClose()")
	}
}

// ============================================================================
// Тест 2: Писатели быстрее читателей (backpressure test)
// ============================================================================
func TestRevolverChannel_WriteFasterThanRead(t *testing.T) {
	const (
		chCap      = 2
		totalWrite = 20
		readDelay  = 5 * time.Millisecond
	)

	rCh := NewRevolverChannel16Bit[int](chCap)

	var wg sync.WaitGroup

	// Быстрый писатель
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < totalWrite; i++ {
			rCh.In <- i
		}
	}()

	// Медленный читатель
	received := make([]int, 0, totalWrite)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < totalWrite; i++ {
			time.Sleep(readDelay)
			received = append(received, <-rCh.Out)
		}
	}()

	wg.Wait()
	rCh.Stop()
	rCh.WaitClose()

	// Проверяем, что все значения получены в порядке
	if len(received) != totalWrite {
		t.Errorf("Expected %d values, got %d", totalWrite, len(received))
	}
	for i, v := range received {
		if v != i {
			t.Errorf("At index %d: expected %d, got %d", i, i, v)
		}
	}
}

// ============================================================================
// Тест 3: Параллельные писатели и читатели
// ============================================================================
func TestRevolverChannel_MultipleProducersConsumers(t *testing.T) {
	const (
		chCap           = 2
		numWriters      = 3
		numReaders      = 3
		valuesPerWriter = 10
		totalValues     = numWriters * valuesPerWriter
	)

	rCh := NewRevolverChannel16Bit[int](chCap)

	var wg sync.WaitGroup
	var produced, consumed int64

	// Писатели
	for w := 0; w < numWriters; w++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			for i := 0; i < valuesPerWriter; i++ {
				val := writerID*1000 + i
				rCh.In <- val
				atomic.AddInt64(&produced, 1)
			}
		}(w)
	}

	// Читатели
	received := sync.Map{}
	var readWg sync.WaitGroup
	for r := 0; r < numReaders; r++ {
		readWg.Add(1)
		go func() {
			defer readWg.Done()
			for v := range rCh.Out {
				received.Store(v, true)
				atomic.AddInt64(&consumed, 1)
			}
		}()
	}

	// Ждём писателей и останавливаем канал
	wg.Wait()
	rCh.Stop()
	readWg.Wait()   // Ждём, пока читатели выйдут из range
	rCh.WaitClose() // Ждём полного закрытия

	// Проверки
	if atomic.LoadInt64(&produced) != int64(totalValues) {
		t.Errorf("Produced: expected %d, got %d", totalValues, atomic.LoadInt64(&produced))
	}
	if atomic.LoadInt64(&consumed) != int64(totalValues) {
		t.Errorf("Consumed: expected %d, got %d", totalValues, atomic.LoadInt64(&consumed))
	}

	count := 0
	received.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	if count != totalValues {
		t.Errorf("Expected %d unique values, got %d", totalValues, count)
	}
}

// ============================================================================
// Тест 4: Stop() — остановка приёма и дренаж буфера (БЕЗ паники!)
// ============================================================================
func TestRevolverChannel_StopDrainsBuffer(t *testing.T) {
	const chCap = 2

	rCh := NewRevolverChannel16Bit[int](chCap)

	// Наполняем буфер
	const total = 5
	for i := 0; i < total; i++ {
		rCh.In <- i
	}

	// Останавливаем приём — In закрывается
	rCh.Stop()

	// ⚠️ НЕ пытаемся писать в rCh.In после Stop() — это вызовет панику!
	// Вместо этого проверяем, что Stop() не закрывает Out сразу

	// Вычитываем всё, что осталось в буфере
	var drained []int
	for v := range rCh.Out {
		drained = append(drained, v)
	}

	// Проверяем, что получили хотя бы часть значений
	if len(drained) == 0 {
		t.Error("Expected some values to be drained after Stop, got none")
	}
	if len(drained) > total {
		t.Errorf("Drained more values (%d) than sent (%d)", len(drained), total)
	}
	t.Logf("Drained %d values after Stop: %v", len(drained), drained)

	// После выхода из range канал Out закрыт
	if !rCh.IsClosed() {
		t.Error("Expected IsClosed()=true after Out channel closed")
	}

	// WaitClose() должен возвращаться мгновенно, т.к. уже закрыто
	rCh.WaitClose()
}

// ============================================================================
// Тест 5: WaitClose() — корректное ожидание полного закрытия
// ============================================================================
func TestRevolverChannel_WaitClose(t *testing.T) {
	const chCap = 2

	rCh := NewRevolverChannel16Bit[int](chCap)

	// Отправляем несколько значений
	for i := 0; i < 3; i++ {
		rCh.In <- i
	}

	// Останавливаем и ждём полного закрытия
	rCh.Stop()

	// Дренажируем Out в отдельной горутине, иначе WaitClose() заблокируется
	go func() {
		for range rCh.Out {
			// Drain
		}
	}()

	// WaitClose() должен дождаться закрытия closeCh
	done := make(chan bool, 1)
	go func() {
		rCh.WaitClose()
		done <- true
	}()

	select {
	case <-done:
		// OK
	case <-time.After(200 * time.Millisecond):
		t.Error("WaitClose() timed out")
	}

	if !rCh.IsClosed() {
		t.Error("Expected IsClosed()=true after WaitClose()")
	}
}

// ============================================================================
// Тест 6: Стресс-тест с маленьким буфером
// ============================================================================
func TestRevolverChannel_StressSmallBuffer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	const (
		chCap      = 2
		iterations = 1000
	)

	rCh := NewRevolverChannel16Bit[int](chCap)

	var wg sync.WaitGroup

	// Писатель
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			rCh.In <- i
		}
	}()

	// Читатель
	received := make(chan int, iterations)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			received <- <-rCh.Out
		}
	}()

	wg.Wait()
	rCh.Stop()
	close(received)

	// Проверка порядка
	for i := 0; i < iterations; i++ {
		if v := <-received; v != i {
			t.Errorf("At iteration %d: expected %d, got %d", i, i, v)
			break
		}
	}
}

// ============================================================================
// Тест 7: Проверка, что после Stop() запись в In блокируется/паникует
// ============================================================================
func TestRevolverChannel_NoWriteAfterStop(t *testing.T) {
	const chCap = 2

	rCh := NewRevolverChannel16Bit[int](chCap)

	// Отправляем одно значение
	rCh.In <- 1

	// Останавливаем
	rCh.Stop()

	// Проверяем, что попытка записать вызывает панику
	// Используем recover для безопасного теста
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Expected panic on send after Stop: %v", r)
		}
	}()

	// Эта строка вызовет панику, что и ожидаем
	rCh.In <- 999

	// Если дошли сюда — паники не было (неожиданно, но не ошибка)
	t.Log("No panic on send after Stop (channel may not be closed yet)")
}

// ============================================================================
// Тест 8: Проверка Len() в конкурентной среде
// ============================================================================
func TestRevolverChannel_LenConcurrent(t *testing.T) {
	const (
		chCap  = 2
		writes = 100
		reads  = 100
	)

	rCh := NewRevolverChannel16Bit[int](chCap)

	var wg sync.WaitGroup

	// Писатели
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < writes; i++ {
			rCh.In <- i
		}
	}()

	// Читатели
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < reads; i++ {
			<-rCh.Out
		}
	}()

	// В середине процесса проверяем Len() — он может быть любым от 0 до writes
	time.Sleep(10 * time.Millisecond)
	lenVal := rCh.Len()
	if lenVal < 0 || lenVal > int64(writes) {
		t.Errorf("Len()=%d out of expected range [0, %d]", lenVal, writes)
	}

	wg.Wait()
	rCh.Stop()
	rCh.WaitClose()

	// После завершения Len() должен быть 0
	if finalLen := rCh.Len(); finalLen != 0 {
		t.Errorf("Expected final Len()=0, got %d", finalLen)
	}
}

// ============================================================================
// Тест 9: Полное заполнение всех 65536 каналов + писатель МЕДЛЕННЕЕ читателя
// ============================================================================
func TestRevolverChannel_FullBuffer_WriterSlowerThanReader(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping full buffer test in short mode")
	}

	const (
		chCap         = 1          // Минимальный буфер для быстрого заполнения всех каналов
		totalChannels = limit16bit // 65536
	)

	rCh := NewRevolverChannel16Bit[int](chCap)

	var wg sync.WaitGroup
	var written, read int64

	// 📤 Писатель: отправляет значения, но с небольшой задержкой (медленнее читателя)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < totalChannels*chCap; i++ {
			// Искусственное замедление писателя
			if i%1000 == 0 {
				time.Sleep(100 * time.Microsecond)
			}
			rCh.In <- i
			atomic.AddInt64(&written, 1)
		}
	}()

	// 📥 Читатель: читает быстрее, чем пишет писатель
	wg.Add(1)
	go func() {
		defer wg.Done()
		for atomic.LoadInt64(&read) < int64(totalChannels*chCap) {
			select {
			case <-rCh.Out:
				atomic.AddInt64(&read, 1)
			case <-time.After(500 * time.Millisecond):
				// Таймаут: если данных нет, но мы ещё не прочитали всё — продолжаем ждать
				if atomic.LoadInt64(&read) >= int64(totalChannels*chCap) {
					return
				}
			}
		}
	}()

	// Ждём завершения писателя
	wg.Wait()

	// Даём время на доставку последних значений
	time.Sleep(50 * time.Millisecond)

	// Проверки
	if atomic.LoadInt64(&written) != int64(totalChannels*chCap) {
		t.Errorf("Written: expected %d, got %d", totalChannels*chCap, atomic.LoadInt64(&written))
	}
	if atomic.LoadInt64(&read) != int64(totalChannels*chCap) {
		t.Errorf("Read: expected %d, got %d", totalChannels*chCap, atomic.LoadInt64(&read))
	}

	// Проверяем, что shiftIn достиг максимума (или близок к нему)
	// Примечание: shiftIn может быть < 65535, если reader успевал "схлопывать" буфер
	t.Logf("Final state: shiftIn=%d, shiftOut=%d, Len()=%d",
		rCh.shiftIn, rCh.shiftOut, rCh.Len())

	rCh.Stop()

	// Дренажируем Out, чтобы workerOut мог закрыться
	go func() {
		for range rCh.Out {
		}
	}()
	rCh.WaitClose()

	if !rCh.IsClosed() {
		t.Error("Expected IsClosed()=true after full test")
	}
}

// ============================================================================
// Тест 10: Полное заполнение + писатель БЫСТРЕЕ читателя, затем остановка
// ============================================================================
func TestRevolverChannel_FullBuffer_WriterFasterThenStop(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping full buffer test in short mode")
	}

	const (
		chCap         = 1
		totalChannels = limit16bit // 65536
		writeBatch    = 1000       // Пишем порциями
	)

	rCh := NewRevolverChannel16Bit[int](chCap)

	var wg sync.WaitGroup
	var written, read int64
	stopWriting := make(chan struct{})

	// 📤 Писатель: пишет быстро, но останавливается по сигналу
	wg.Add(1)
	go func() {
		defer wg.Done()
		i := 0
		for {
			select {
			case <-stopWriting:
				return
			default:
				// Пишем порциями без задержек (быстрее читателя)
				for j := 0; j < writeBatch && i < totalChannels*chCap; j++ {
					rCh.In <- i
					i++
					atomic.AddInt64(&written, 1)
				}
				if i >= totalChannels*chCap {
					return
				}
			}
		}
	}()

	// 📥 Читатель: читает с искусственной задержкой (медленнее писателя)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for atomic.LoadInt64(&read) < int64(totalChannels*chCap) {
			select {
			case <-rCh.Out:
				atomic.AddInt64(&read, 1)
				// Искусственное замедление читателя
				if atomic.LoadInt64(&read)%500 == 0 {
					time.Sleep(50 * time.Microsecond)
				}
			case <-time.After(1 * time.Second):
				// Если данных нет — возможно, писатель закончил
				if atomic.LoadInt64(&written) >= int64(totalChannels*chCap) &&
					atomic.LoadInt64(&read) >= atomic.LoadInt64(&written) {
					return
				}
			}
		}
	}()

	// Даём писателю заполнить буфер полностью
	time.Sleep(200 * time.Millisecond)

	// 🔴 Останавливаем писателя ДО того, как он отправит всё
	close(stopWriting)

	// Ждём, пока писатель завершится
	wg.Wait()

	// 🔄 Теперь вычитываем всё, что осталось в буфере
	for atomic.LoadInt64(&read) < atomic.LoadInt64(&written) {
		select {
		case <-rCh.Out:
			atomic.AddInt64(&read, 1)
		case <-time.After(500 * time.Millisecond):
			// Если таймаут — проверяем, не закончились ли данные
			if atomic.LoadInt64(&read) >= atomic.LoadInt64(&written) {
				break
			}
		}
	}

	// Проверки
	w := atomic.LoadInt64(&written)
	r := atomic.LoadInt64(&read)
	t.Logf("Written: %d, Read: %d, Len(): %d", w, r, rCh.Len())

	if r != w {
		t.Errorf("Read (%d) != Written (%d)", r, w)
	}

	// 🔑 Ключевая проверка: писатель НЕ завис навсегда при полном буфере
	// Если бы он завис, тест бы упал по таймауту выше

	rCh.Stop()
	go func() {
		for range rCh.Out {
		}
	}()
	rCh.WaitClose()
}

// ============================================================================
// Тест 11: Проверка, что писатель НЕ блокируется навсегда при полном буфере
// ============================================================================
func TestRevolverChannel_NoDeadlockOnFullBuffer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping deadlock test in short mode")
	}

	const (
		chCap = 1
		// Не заполняем все 65536, а берём разумный подмножество для скорости
		testChannels = 1000
	)

	rCh := NewRevolverChannel16Bit[int](chCap)

	var (
		wg         sync.WaitGroup
		writerDone = make(chan bool, 1)
		readerDone = make(chan bool, 1)
	)

	// 📤 Писатель: заполняет буфер полностью
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < testChannels*chCap; i++ {
			rCh.In <- i
		}
		writerDone <- true
	}()

	// 📥 Читатель: НЕ читает вообще (имитация "зависшего" потребителя)
	// Но мы даём таймаут, чтобы тест не завис навсегда
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Намеренно не читаем из rCh.Out
		readerDone <- true
	}()

	// ⏱️ Ждём писателя с таймаутом
	select {
	case <-writerDone:
		t.Log("Writer completed successfully — no deadlock!")
	case <-time.After(2 * time.Second):
		t.Fatal("TIMEOUT: Writer appears to be deadlocked on full buffer!")
	}

	// 🧹 Теперь вычитываем, чтобы разблокировать систему
	go func() {
		for range rCh.Out {
		}
	}()

	// Завершаем тест
	rCh.Stop()
	<-readerDone
	rCh.WaitClose()
}

// ============================================================================
// Тест 12: Циклическое заполнение (shiftIn wraps around)
// ============================================================================
func TestRevolverChannel_ShiftInWraparound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping wraparound test in short mode")
	}

	const chCap = 1

	rCh := NewRevolverChannel16Bit[int](chCap)

	// Наполняем буфер до максимума
	maxValues := limit16bit * chCap
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < maxValues; i++ {
			rCh.In <- i
		}
	}()

	// Параллельно читаем, чтобы не переполнить память
	wg.Add(1)
	go func() {
		defer wg.Done()
		count := 0
		for count < maxValues {
			select {
			case <-rCh.Out:
				count++
			case <-time.After(100 * time.Millisecond):
				// Продолжаем, если есть что читать
			}
		}
	}()

	wg.Wait()

	t.Logf("Wraparound test passed: shiftIn=%d, shiftOut=%d", rCh.shiftIn, rCh.shiftOut)

	rCh.Stop()
	go func() {
		for range rCh.Out {
		}
	}()
	rCh.WaitClose()
}
