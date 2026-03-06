# Circle sector control

A lock-free, concurrent-safe circular buffer controller designed for multi-worker scenarios. It manages head and tail pointers that move independently in a circular fashion, ensuring the tail never overtakes the head.

Ideal for use cases like `RevolverChannel`, where multiple workers need to coordinate access to a rotating set of resources without hard limits on pointer types (e.g., moving beyond `uint8` or `uint16`).

## Installation

```bash
go get github.com/claygod/tools/circle_sector_control
```

## PI Reference

### Types

#### `CircleSectorControl`

The main structure that manages circular buffer state.

```go
type CircleSectorControl struct {
    // Internal fields (not exported)
    cap  int64
    head int64
    tail int64
}
```

### Constructor

#### `NewCircleSectorControl(cap int64) *CircleSectorControl`

Creates a new `CircleSectorControl` instance.

**Parameters:**
- `cap` — Total number of sectors in the circular buffer (must be positive)

**Returns:**
- `*CircleSectorControl` — Pointer to the new instance

**Panics:**
- If `cap <= 0`

**Example:**
```go
c := circle_sector_control.NewCircleSectorControl(100)
```

### Public Methods

#### `HeadForward() (int64, bool)`

Moves the `head` pointer forward by one position (circularly).

**Returns:**
- `int64` — New head position (or current if movement failed)
- `bool` — `true` if successful, `false` if head would catch up to tail

**Behavior:**
- Blocks head from overtaking tail (ensures at least 1 sector remains active)
- Uses atomic CAS for thread-safety
- Retries automatically on contention

**Example:**
```go
newHead, ok := c.HeadForward()
if !ok {
    fmt.Println("Head cannot move - would catch tail")
}
```

#### `TailForward() (int64, bool)`

Moves the `tail` pointer forward by one position (circularly).

**Returns:**
- `int64` — New tail position (or current if movement failed)
- `bool` — `true` if successful, `false` if tail is at head (minimum state)

**Behavior:**
- Prevents tail from exceeding head (ensures at least 1 sector remains active)
- Uses atomic CAS for thread-safety
- Retries automatically on contention

**Example:**
```go
newTail, ok := c.TailForward()
if !ok {
    fmt.Println("Tail cannot move - already at head")
}
```

#### `Head() int64`

Returns the current head position (atomic read).

**Returns:**
- `int64` — Current head index

**Example:**
```go
head := c.Head()
fmt.Printf("Head position: %d\n", head)
```

#### `Tail() int64`

Returns the current tail position (atomic read).

**Returns:**
- `int64` — Current tail index

**Example:**
```go
tail := c.Tail()
fmt.Printf("Tail position: %d\n", tail)
```

#### `Utilization() float64`

Calculates the percentage of sectors currently in use (0–100%).

**Returns:**
- `float64` — Utilization percentage

**Behavior:**
- Accounts for wrap-around (when `head < tail`)
- Guarantees result is between 0 and 100
- At minimum state (`head == tail`), returns `1/cap * 100` (at least 1 sector active)

**Example:**
```go
util := c.Utilization()
fmt.Printf("Buffer utilization: %.2f%%\n", util)
```

## Usage Examples

### Example 1: Basic Single-Threaded Usage

```go
package main

import (
    "fmt"
    "github.com/claygod/tools/circle_sector_control"
)

func main() {
    c := circle_sector_control.NewCircleSectorControl(10)

    fmt.Printf("Initial: head=%d, tail=%d, util=%.2f%%\n", 
        c.Head(), c.Tail(), c.Utilization())

    // Move head forward
    for i := 0; i < 5; i++ {
        head, ok := c.HeadForward()
        if ok {
            fmt.Printf("Head moved to %d\n", head)
        }
    }

    // Move tail forward
    for i := 0; i < 3; i++ {
        tail, ok := c.TailForward()
        if ok {
            fmt.Printf("Tail moved to %d\n", tail)
        }
    }

    fmt.Printf("Final: head=%d, tail=%d, util=%.2f%%\n", 
        c.Head(), c.Tail(), c.Utilization())
}
```

**Output:**
```
Initial: head=0, tail=0, util=10.00%
Head moved to 1
Head moved to 2
Head moved to 3
Head moved to 4
Head moved to 5
Tail moved to 1
Tail moved to 2
Tail moved to 3
Final: head=5, tail=3, util=30.00%
```

### Example 2: Concurrent Workers (Producer/Consumer)

```go
package main

import (
    "fmt"
    "sync"
    "time"
    "github.com/claygod/tools/circle_sector_control"
)

func main() {
    c := circle_sector_control.NewCircleSectorControl(100)
    var wg sync.WaitGroup

    // Worker 1: Produces (moves head)
    wg.Add(1)
    go func() {
        defer wg.Done()
        for i := 0; i < 50; i++ {
            if _, ok := c.HeadForward(); ok {
                fmt.Printf("[Producer] Head advanced\n")
            } else {
                fmt.Printf("[Producer] Blocked - buffer full\n")
            }
            time.Sleep(time.Millisecond)
        }
    }()

    // Worker 2: Consumes (moves tail)
    wg.Add(1)
    go func() {
        defer wg.Done()
        for i := 0; i < 50; i++ {
            if _, ok := c.TailForward(); ok {
                fmt.Printf("[Consumer] Tail advanced\n")
            } else {
                fmt.Printf("[Consumer] Blocked - buffer empty\n")
            }
            time.Sleep(time.Millisecond)
        }
    }()

    // Monitor utilization
    wg.Add(1)
    go func() {
        defer wg.Done()
        for i := 0; i < 20; i++ {
            fmt.Printf("[Monitor] Utilization: %.2f%%\n", c.Utilization())
            time.Sleep(5 * time.Millisecond)
        }
    }()

    wg.Wait()
    fmt.Printf("Final: head=%d, tail=%d, util=%.2f%%\n", 
        c.Head(), c.Tail(), c.Utilization())
}
```

### Example 3: Integration with RevolverChannel

```go
package main

import (
    "github.com/claygod/tools/circle_sector_control"
)

type RevolverChannel struct {
    control *circle_sector_control.CircleSectorControl
    channels []chan int
}

func NewRevolverChannel(cap int) *RevolverChannel {
    control := circle_sector_control.NewCircleSectorControl(int64(cap))
    channels := make([]chan int, cap)
    for i := 0; i < cap; i++ {
        channels[i] = make(chan int, 10)
    }
    return &RevolverChannel{
        control:  control,
        channels: channels,
    }
}

func (r *RevolverChannel) Send(val int) bool {
    head, ok := r.control.HeadForward()
    if !ok {
        return false // Buffer full
    }
    // Use (head - 1) as the actual channel index
    idx := head - 1
    if idx < 0 {
        idx = int64(len(r.channels)) - 1
    }
    r.channels[idx] <- val
    return true
}

func (r *RevolverChannel) Receive() (int, bool) {
    tail, ok := r.control.TailForward()
    if !ok {
        return 0, false // Buffer empty
    }
    // Use (tail - 1) as the actual channel index
    idx := tail - 1
    if idx < 0 {
        idx = int64(len(r.channels)) - 1
    }
    val := <-r.channels[idx]
    return val, true
}

func (r *RevolverChannel) Utilization() float64 {
    return r.control.Utilization()
}
```

### Example 4: Wrap-Around Behavior

```go
package main

import (
    "fmt"
    "github.com/claygod/tools/circle_sector_control"
)

func main() {
    c := circle_sector_control.NewCircleSectorControl(5)

    // Fill the buffer
    for i := 0; i < 4; i++ {
        c.HeadForward()
    }
    // head=4, tail=0

    fmt.Printf("Before wrap: head=%d, tail=%d\n", c.Head(), c.Tail())

    // Head wraps around
    head, ok := c.HeadForward()
    fmt.Printf("After head wrap: head=%d, ok=%v\n", head, ok)
    // head=0, tail=0 (would catch tail, so may fail depending on state)

    // Move tail to allow more head movement
    c.TailForward()
    c.TailForward()

    fmt.Printf("After tail move: head=%d, tail=%d, util=%.2f%%\n", 
        c.Head(), c.Tail(), c.Utilization())
}
```

## ⚠️ Important Notes

| Aspect | Description |
|--------|-------------|
| **Thread-Safety** | All public methods are safe for concurrent use by multiple goroutines |
| **Minimum State** | At least 1 sector is always active (`head == tail` = 1 sector in use) |
| **No Blocking** | Methods return `false` immediately if movement is not possible (non-blocking) |
| **Wrap-Around** | Pointers automatically wrap to 0 when reaching `cap` |
| **Memory** | This struct only manages pointers — actual data storage is external |

## Running Tests

```bash
# Run all tests
go test -v ./...

# Run with race detector
go test -race -v ./...

# Run specific test
go test -v -run TestUtilization

# Stress test (may be skipped in short mode)
go test -v -run TestStress
```

## Benchmarks

cpu: Intel(R) Core(TM) i7-6700T CPU @ 2.80GHz
BenchmarkHeadForward_SingleThread-8              	879956611	         1.372 ns/op
BenchmarkTailForward_SingleThread-8              	806843685	         1.477 ns/op
BenchmarkUtilization_SingleThread-8              	1000000000	         0.5910 ns/op
BenchmarkHeadForward_Concurrent-8                	1000000000	         0.3182 ns/op
BenchmarkTailForward_Concurrent-8                	1000000000	         0.3180 ns/op
BenchmarkHeadTail_Concurrent-8                   	18304912	        65.72 ns/op
BenchmarkHeadTail_Concurrent_Separate/head_workers-8         	1000000000	         0.3730 ns/op
BenchmarkHeadTail_Concurrent_Separate/tail_workers-8         	1000000000	         0.3184 ns/op
BenchmarkHeadTail_Concurrent_WithUtilization-8               	17982464	        69.08 ns/op
BenchmarkHeadForward_CapSmall-8                              	1000000000	         0.3184 ns/op
BenchmarkHeadForward_CapLarge-8                              	1000000000	         0.3805 ns/op
BenchmarkHeadTail_Concurrent_CapVariations/cap_10-8          	18368478	        66.50 ns/op
BenchmarkHeadTail_Concurrent_CapVariations/cap_100-8         	17817634	        66.28 ns/op
BenchmarkHeadTail_Concurrent_CapVariations/cap_1000-8        	18276157	        65.90 ns/op
BenchmarkHeadTail_Concurrent_CapVariations/cap_10000-8       	18380395	        68.91 ns/op
BenchmarkHeadForward_WithContention-8                        	1000000000	         0.3179 ns/op
BenchmarkTailForward_WithContention-8                        	1000000000	         0.3186 ns/op
BenchmarkAtomic_LoadInt64-8                                  	1000000000	         0.2968 ns/op
BenchmarkAtomic_CAS_Int64-8                                  	127667815	         9.408 ns/op
BenchmarkCircleSector_FullCycle-8                            	  546453	      2003 ns/op
BenchmarkRevolverChannel_Typical-8                           	  468108	      2476 ns/op
BenchmarkRevolverChannel_HighLoad-8                          	18232840	        67.15 ns/op
BenchmarkUtilization_DifferentStates/min-8                   	1000000000	         0.5884 ns/op
BenchmarkUtilization_DifferentStates/half-8                  	1000000000	         0.5894 ns/op
BenchmarkUtilization_DifferentStates/full-8                  	1000000000	         0.5949 ns/op
BenchmarkUtilization_DifferentStates/wrap-8                  	1000000000	         0.5966 ns/op
BenchmarkHeadTail_ContentionProfile-8                        	16729848	        73.39 ns/op
--- BENCH: BenchmarkHeadTail_ContentionProfile-8
    circle_sector_control_bench_test.go:365: success=0, fail=1, fail_rate=100.00%
    circle_sector_control_bench_test.go:365: success=99, fail=1, fail_rate=1.00%
    circle_sector_control_bench_test.go:365: success=9994, fail=6, fail_rate=0.06%
    circle_sector_control_bench_test.go:365: success=999963, fail=37, fail_rate=0.00%
    circle_sector_control_bench_test.go:365: success=16729383, fail=465, fail_rate=0.00%
    
**Analysis**:

- Execution code for 2 workloads + 1 CAS + operation logic
- Theoretical minimum: ~2×0.3 + 9.4 = ~10 ns
- Accepted: ~65 ns
- The ~55 ns difference is the price of cache coherence (cache synchronization between cores).

This is normal for concurrent code without locks. You're paying for the ability to run concurrently without mutexes.

**Conclusion**:

65 ns for a concurrent lock-free operation isn't "slow," it's "the price of parallelism."

You get:

✅ No deadlocks
✅ Scalability to multiple cores
✅ Predictable latency (no mutex queues)

Cost:

⚠️ ~6× cache coherency overhead (compared to single-threaded mode)

### License

Copyright © 2026 Eduard Sesigin. All rights reserved.
Contact: [claygod@yandex.ru](mailto:claygod@yandex.ru)