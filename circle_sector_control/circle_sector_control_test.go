package circle_sector_control

// Circle sector control
// Tests 1
// Copyright © 2026 Eduard Sesigin. All rights reserved. Contacts: <claygod@yandex.ru>

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ==================== Тесты Utilization ====================

func TestUtilization_Basic(t *testing.T) {
	tests := []struct {
		name     string
		cap      int64
		head     int64
		tail     int64
		expected float64
	}{
		{"min_state_cap10", 10, 0, 0, 10},  // 1 секция из 10 = 10%
		{"min_state_cap100", 100, 0, 0, 1}, // 1 секция из 100 = 1%
		{"half_full", 10, 4, 0, 50},        // 5 секций из 10 = 50%
		{"full", 10, 9, 0, 100},            // 10 секций из 10 = 100%
		{"cap1_always_full", 1, 0, 0, 100}, // 1 секция из 1 = 100%
		{"cap2_min", 2, 0, 0, 50},          // 1 секция из 2 = 50%
		{"cap2_full", 2, 1, 0, 100},        // 2 секции из 2 = 100%
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &CircleSectorControl{cap: tt.cap, head: tt.head, tail: tt.tail}
			util := c.Utilization()
			if util != tt.expected {
				t.Errorf("got %.2f%%, want %.2f%%", util, tt.expected)
			}
		})
	}
}

func TestUtilization_WrapAround(t *testing.T) {
	// head < tail — секции перешли через границу cap
	tests := []struct {
		name     string
		cap      int64
		head     int64
		tail     int64
		expected float64
	}{
		{"wrap_50_percent", 10, 2, 8, 50}, // 10-8+2+1 = 5 секций
		{"wrap_60_percent", 10, 0, 5, 60}, // 10-5+0+1 = 6 секций
		{"wrap_40_percent", 10, 1, 8, 40}, // 10-8+1+1 = 4 секции
		{"wrap_near_full", 10, 6, 7, 100}, // 10-7+6+1 = 10 секций
		{"wrap_cap20", 20, 3, 15, 45},     // 20-15+3+1 = 9 секций = 45%
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &CircleSectorControl{cap: tt.cap, head: tt.head, tail: tt.tail}
			util := c.Utilization()
			if util != tt.expected {
				t.Errorf("cap=%d, head=%d, tail=%d: got %.2f%%, want %.2f%%",
					tt.cap, tt.head, tt.tail, util, tt.expected)
			}
		})
	}
}

func TestUtilization_NeverExceeds100(t *testing.T) {
	c := &CircleSectorControl{cap: 10, head: 9, tail: 0}
	util := c.Utilization()
	if util > 100 {
		t.Errorf("utilization %.2f%% exceeds 100%%", util)
	}
}

func TestUtilization_NeverBelowMinimum(t *testing.T) {
	// Минимальная утилизация = 1/cap * 100 (когда head == tail)
	caps := []int64{1, 2, 5, 10, 100}

	for _, cap := range caps {
		c := &CircleSectorControl{cap: cap, head: 0, tail: 0}
		util := c.Utilization()
		expected := 100.0 / float64(cap)
		if util < expected {
			t.Errorf("cap=%d: utilization %.2f%% below minimum %.2f%%",
				cap, util, expected)
		}
	}
}

// ==================== Тесты HeadForward ====================

func TestHeadForward_Basic(t *testing.T) {
	c := NewCircleSectorControl(10)

	// Первое движение head
	newHead, ok := c.HeadForward()
	if !ok {
		t.Error("HeadForward should succeed on first call")
	}
	if newHead != 1 {
		t.Errorf("expected head=1, got %d", newHead)
	}

	// Ещё несколько движений
	for i := int64(2); i <= 5; i++ {
		newHead, ok = c.HeadForward()
		if !ok {
			t.Errorf("HeadForward should succeed at iteration %d", i)
		}
		if newHead != i {
			t.Errorf("expected head=%d, got %d", i, newHead)
		}
	}
}

func TestHeadForward_CannotCatchTail(t *testing.T) {
	c := NewCircleSectorControl(5)

	// Устанавливаем tail = 3 (через движение head, потом tail догонит)
	for i := 0; i < 3; i++ {
		c.HeadForward()
	}
	for i := 0; i < 2; i++ {
		c.TailForward()
	}
	// head = 3, tail = 2

	// Двигаем head до 4
	c.HeadForward()
	// head = 4, tail = 2

	// Следующее движение head: newHead = 0, tail = 2 → OK
	c.HeadForward()
	// head = 0, tail = 2

	// Двигаем tail до 0
	c.TailForward() // tail = 3
	c.TailForward() // tail = 4
	c.TailForward() // tail = 0
	// head = 0, tail = 0 (минимальное состояние)

	// Теперь head не может двигаться (newHead = 1, но tail = 0... подождите)
	// На самом деле head=0, tail=0, newHead=1 != tail=0 → OK

	// Давайте создадим ситуацию где head упрётся в tail
	c2 := NewCircleSectorControl(3)
	c2.HeadForward() // head=1, tail=0
	c2.HeadForward() // head=2, tail=0
	// head=2, tail=0

	// Теперь tail догоняет
	c2.TailForward() // tail=1
	c2.TailForward() // tail=2
	// head=2, tail=2 (минимальное состояние)

	// Head не может двигаться (newHead=0 != tail=2... тоже OK)

	// Создаём ситуацию head+1 == tail
	c3 := NewCircleSectorControl(3)
	c3.HeadForward() // head=1, tail=0
	// head=1, tail=0, newHead=2 != tail=0 → OK

	// А теперь tail=2
	c3.TailForward() // tail=1
	// head=1, tail=1 → минимальное состояние

	_, ok := c3.HeadForward()
	// newHead = 2, tail = 1 → OK, должно сработать

	// Настоящий тест: tail опережает head на 1
	c4 := NewCircleSectorControl(3)
	c4.HeadForward() // head=1, tail=0
	c4.HeadForward() // head=2, tail=0
	c4.TailForward() // tail=1
	c4.TailForward() // tail=2
	// head=2, tail=2

	// HeadForward: newHead=0, tail=2 → OK
	newHead, ok := c4.HeadForward()
	if !ok {
		t.Error("HeadForward should succeed when newHead != tail")
	}
	if newHead != 0 {
		t.Errorf("expected head=0 after wrap, got %d", newHead)
	}

	// Теперь head=0, tail=2
	// HeadForward: newHead=1, tail=2 → OK
	newHead, ok = c4.HeadForward()
	if !ok {
		t.Error("HeadForward should succeed")
	}
	if newHead != 1 {
		t.Errorf("expected head=1, got %d", newHead)
	}

	// Теперь head=1, tail=2
	// HeadForward: newHead=2, tail=2 → FAIL (newHead == tail)
	_, ok = c4.HeadForward()
	if ok {
		t.Error("HeadForward should fail when newHead == tail")
	}
}

func TestHeadForward_WrapAround(t *testing.T) {
	c := NewCircleSectorControl(5)

	// Сначала двигаем tail, чтобы освободить место для wrap
	c.HeadForward() // head=1, tail=0
	c.HeadForward() // head=2, tail=0
	c.TailForward() // head=2, tail=1 ← освобождаем сектор 0

	// Теперь head может пройти через границу
	c.HeadForward() // head=3
	c.HeadForward() // head=4

	// Следующий wrap: newHead=0, tail=1 → 0≠1 → OK
	newHead, ok := c.HeadForward()
	if !ok {
		t.Error("HeadForward should succeed at wrap when tail != 0")
	}
	if newHead != 0 {
		t.Errorf("expected head=0 after wrap, got %d", newHead)
	}
}

func TestHeadForward_ReturnsCorrectValue(t *testing.T) {
	c := NewCircleSectorControl(5)

	// При успехе возвращается новое значение head
	newHead, ok := c.HeadForward()
	if !ok || newHead != 1 {
		t.Errorf("expected (1, true), got (%d, %v)", newHead, ok)
	}

	// При неудаче возвращается старое значение head
	c2 := NewCircleSectorControl(2)
	c2.HeadForward() // head=1, tail=0
	// head=1, tail=0, newHead=0 != tail → OK
	c2.HeadForward() // head=0, tail=0 (минимальное)
	// head=0, tail=0, newHead=1 != tail → OK

	// Заполняем до отказа
	c3 := NewCircleSectorControl(3)
	c3.HeadForward() // head=1
	c3.HeadForward() // head=2
	// head=2, tail=0

	// Tail догоняет
	c3.TailForward() // tail=1
	c3.TailForward() // tail=2
	// head=2, tail=2 (минимальное)

	// Head может двигаться (newHead=0 != tail=2)
	c3.HeadForward() // head=0
	// head=0, tail=2

	c3.HeadForward() // head=1
	// head=1, tail=2

	// Теперь newHead=2 == tail=2 → FAIL
	oldHead := c3.Head()
	returnedHead, ok := c3.HeadForward()
	if ok {
		t.Error("HeadForward should fail")
	}
	if returnedHead != oldHead {
		t.Errorf("on failure, should return old head %d, got %d", oldHead, returnedHead)
	}
}

// ==================== Тесты TailForward ====================

func TestTailForward_Basic(t *testing.T) {
	c := NewCircleSectorControl(10)

	// Сначала двигаем head, чтобы tail мог двигаться
	c.HeadForward()
	c.HeadForward()
	// head = 2, tail = 0

	newTail, ok := c.TailForward()
	if !ok {
		t.Error("TailForward should succeed when head > tail")
	}
	if newTail != 1 {
		t.Errorf("expected tail=1, got %d", newTail)
	}
}

func TestTailForward_CannotExceedHead(t *testing.T) {
	// При head == tail (минимальное состояние) tail не может двигаться
	c := NewCircleSectorControl(5)
	// head = 0, tail = 0

	_, ok := c.TailForward()
	if ok {
		t.Error("TailForward should fail when head == tail (minimum state)")
	}

	// Двигаем head вперёд
	c.HeadForward()
	// head = 1, tail = 0

	// Теперь tail может догнать head
	newTail, ok := c.TailForward()
	if !ok {
		t.Error("TailForward should succeed when head > tail")
	}
	if newTail != 1 {
		t.Errorf("expected tail=1, got %d", newTail)
	}
	// head = 1, tail = 1 (снова минимальное состояние)

	// Снова не может двигаться
	_, ok = c.TailForward()
	if ok {
		t.Error("TailForward should fail again when head == tail")
	}
}

func TestTailForward_WrapAround(t *testing.T) {
	c := NewCircleSectorControl(5)

	// Заполняем буфер
	for i := 0; i < 4; i++ {
		c.HeadForward()
	}
	// head = 4, tail = 0

	// Двигаем tail через границу
	for i := 0; i < 4; i++ {
		c.TailForward()
	}
	// tail = 4, head = 4 (минимальное состояние)

	// Tail больше не может двигаться
	_, ok := c.TailForward()
	if ok {
		t.Error("TailForward should fail when tail caught up to head")
	}
}

func TestTailForward_ReturnsCorrectValue(t *testing.T) {
	c := NewCircleSectorControl(5)
	c.HeadForward() // head=1, tail=0

	// При успехе возвращается новое значение tail
	newTail, ok := c.TailForward()
	if !ok || newTail != 1 {
		t.Errorf("expected (1, true), got (%d, %v)", newTail, ok)
	}

	// При неудаче возвращается старое значение tail
	oldTail := c.Tail()
	returnedTail, ok := c.TailForward()
	if ok {
		t.Error("TailForward should fail when head == tail")
	}
	if returnedTail != oldTail {
		t.Errorf("on failure, should return old tail %d, got %d", oldTail, returnedTail)
	}
}

// ==================== Тесты Concurrent Access ====================

func TestConcurrent_HeadTail(t *testing.T) {
	c := NewCircleSectorControl(100)
	var wg sync.WaitGroup
	iterations := 1000

	headMoves := int64(0)
	tailMoves := int64(0)

	wg.Add(2)

	// Воркер 1: двигает head
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			if _, ok := c.HeadForward(); ok {
				atomic.AddInt64(&headMoves, 1)
			}
			time.Sleep(time.Microsecond)
		}
	}()

	// Воркер 2: двигает tail
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			if _, ok := c.TailForward(); ok {
				atomic.AddInt64(&tailMoves, 1)
			}
			time.Sleep(time.Microsecond)
		}
	}()

	wg.Wait()

	head := c.Head()
	tail := c.Tail()

	t.Logf("head=%d, tail=%d, headMoves=%d, tailMoves=%d",
		head, tail, headMoves, tailMoves)
	t.Logf("utilization: %.2f%%", c.Utilization())

	// Утилизация никогда не должна превышать 100%
	if c.Utilization() > 100 {
		t.Errorf("utilization %.2f%% exceeds 100%%", c.Utilization())
	}

	// Утилизация никогда не должна быть ниже минимума
	minUtil := 100.0 / float64(100)
	if c.Utilization() < minUtil {
		t.Errorf("utilization %.2f%% below minimum %.2f%%",
			c.Utilization(), minUtil)
	}
}

func TestConcurrent_UtilizationStable(t *testing.T) {
	c := NewCircleSectorControl(50)
	var wg sync.WaitGroup
	iterations := 500

	wg.Add(3)

	// Воркер 1: двигает head
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			c.HeadForward()
			time.Sleep(time.Microsecond)
		}
	}()

	// Воркер 2: двигает tail
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			c.TailForward()
			time.Sleep(time.Microsecond)
		}
	}()

	// Воркер 3: постоянно проверяет утилизацию
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			util := c.Utilization()
			if util < 0 || util > 100 {
				t.Errorf("invalid utilization: %.2f%%", util)
			}
			time.Sleep(time.Microsecond)
		}
	}()

	wg.Wait()
}

// ==================== Тесты Edge Cases ====================

func TestCap_One(t *testing.T) {
	c := NewCircleSectorControl(1)

	// При cap=1, head=tail=0 — утилизация 100%
	util := c.Utilization()
	if util != 100 {
		t.Errorf("expected 100%% for cap=1, got %.2f%%", util)
	}

	// Head не может двигаться (newHead=0 == tail=0)
	_, ok := c.HeadForward()
	if ok {
		t.Error("HeadForward should fail for cap=1")
	}

	// Tail не может двигаться (head == tail)
	_, ok = c.TailForward()
	if ok {
		t.Error("TailForward should fail for cap=1")
	}
}

func TestCap_Two(t *testing.T) {
	c := NewCircleSectorControl(2)

	// Начальное состояние: head=tail=0 → 1 секция занята = 50%
	util := c.Utilization()
	if util != 50 {
		t.Errorf("expected 50%% for cap=2 head=tail=0, got %.2f%%", util)
	}

	// Двигаем head
	newHead, ok := c.HeadForward()
	if !ok {
		t.Error("HeadForward should succeed for cap=2")
	}
	if newHead != 1 {
		t.Errorf("expected head=1, got %d", newHead)
	}

	// Теперь head=1, tail=0 → 2 секции = 100%
	util = c.Utilization()
	if util != 100 {
		t.Errorf("expected 100%% after head forward, got %.2f%%", util)
	}

	// Head больше не может двигаться (newHead=0 == tail=0)
	_, ok = c.HeadForward()
	if ok {
		t.Error("HeadForward should fail when buffer full")
	}

	// Двигаем tail
	newTail, ok := c.TailForward()
	if !ok {
		t.Error("TailForward should succeed when head > tail")
	}
	if newTail != 1 {
		t.Errorf("expected tail=1, got %d", newTail)
	}

	// head=1, tail=1 → 1 секция = 50%
	util = c.Utilization()
	if util != 50 {
		t.Errorf("expected 50%% after tail forward, got %.2f%%", util)
	}
}

func TestIncrement_WrapAround(t *testing.T) {
	c := &CircleSectorControl{cap: 5}

	tests := []struct {
		input    int64
		expected int64
	}{
		{0, 1},
		{3, 4},
		{4, 0}, // wrap при cap-1
	}

	for _, tt := range tests {
		result := c.increment(tt.input)
		if result != tt.expected {
			t.Errorf("increment(%d) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}

func TestNewCircleSectorControl_InvalidCap(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("NewCircleSectorControl should panic for cap <= 0")
		}
	}()

	NewCircleSectorControl(0)
}

func TestNewCircleSectorControl_NegativeCap(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("NewCircleSectorControl should panic for negative cap")
		}
	}()

	NewCircleSectorControl(-5)
}

func TestHeadTail_Getters(t *testing.T) {
	c := NewCircleSectorControl(10)

	if c.Head() != 0 {
		t.Errorf("expected head=0, got %d", c.Head())
	}
	if c.Tail() != 0 {
		t.Errorf("expected tail=0, got %d", c.Tail())
	}

	c.HeadForward()
	c.HeadForward()
	c.TailForward()

	if c.Head() != 2 {
		t.Errorf("expected head=2, got %d", c.Head())
	}
	if c.Tail() != 1 {
		t.Errorf("expected tail=1, got %d", c.Tail())
	}
}

// ==================== Стресс-тест ====================

func TestStress_Concurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	c := NewCircleSectorControl(1000)
	var wg sync.WaitGroup
	workers := 10
	iterations := 10000

	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				if id%2 == 0 {
					c.HeadForward()
				} else {
					c.TailForward()
				}

				// Периодически проверяем утилизацию
				if j%100 == 0 {
					util := c.Utilization()
					if util < 0 || util > 100 {
						t.Errorf("worker %d: invalid utilization %.2f%%", id, util)
					}
				}
			}
		}(i)
	}

	wg.Wait()

	t.Logf("Final: head=%d, tail=%d, utilization=%.2f%%",
		c.Head(), c.Tail(), c.Utilization())
}

// ==================== Тесты для RevolverChannel сценариев ====================

func TestRevolverChannel_Scenario_EmptyToFull(t *testing.T) {
	// Симуляция сценария RevolverChannel: от 1 канала до максимума
	c := NewCircleSectorControl(10)

	// Начальное состояние: 1 канал активен
	if c.Utilization() != 10 {
		t.Errorf("expected 10%% at start, got %.2f%%", c.Utilization())
	}

	// Добавляем каналы (HeadForward)
	for i := 0; i < 9; i++ {
		_, ok := c.HeadForward()
		if !ok && i < 9 {
			t.Errorf("HeadForward should succeed at iteration %d", i)
		}
	}

	// Теперь должно быть 10 каналов (100%)
	if c.Utilization() != 100 {
		t.Errorf("expected 100%% when full, got %.2f%%", c.Utilization())
	}

	// Head больше не может двигаться
	_, ok := c.HeadForward()
	if ok {
		t.Error("HeadForward should fail when full")
	}

	// Освобождаем каналы (TailForward)
	for i := 0; i < 9; i++ {
		_, ok := c.TailForward()
		if !ok && i < 9 {
			t.Errorf("TailForward should succeed at iteration %d", i)
		}
	}

	// Вернулись к 1 каналу (10%)
	if c.Utilization() != 10 {
		t.Errorf("expected 10%% after drain, got %.2f%%", c.Utilization())
	}
}

func TestRevolverChannel_Scenario_WrapWithMinimumState(t *testing.T) {
	// Сценарий: head и tail проходят через границу cap, сохраняя минимальное состояние
	c := NewCircleSectorControl(5)

	// Заполняем
	for i := 0; i < 4; i++ {
		c.HeadForward()
	}
	// head=4, tail=0, utilization=100%

	if c.Utilization() != 100 {
		t.Errorf("expected 100%% when full, got %.2f%%", c.Utilization())
	}

	// Опустошаем до минимума
	for i := 0; i < 4; i++ {
		c.TailForward()
	}
	// head=4, tail=4, utilization=20% (1 секция из 5)

	if c.Utilization() != 20 {
		t.Errorf("expected 20%% at minimum, got %.2f%%", c.Utilization())
	}

	// Снова заполняем (через wrap)
	for i := 0; i < 4; i++ {
		c.HeadForward()
	}
	// head=3, tail=4, utilization=100%

	if c.Utilization() != 100 {
		t.Errorf("expected 100%% after wrap-fill, got %.2f%%", c.Utilization())
	}
}
