# Revolver Channel

A generic, auto-scaling buffered channel for Go.

For chCap=100 and an expected peak of 5000 messages, RevolverChannel8 is sufficient (5000/100 = 50 slots < 256).
For maximum overload protection, use RevolverChannel16.

## Quick Start

```go
package main

import "github.com/claygod/tools/revolver-channel"

func main() {
    // Create a channel with buffer capacity 100 per internal slot
    ch := revolver_channel.NewRevolverChannel16Bit[string](100)
    
    // Send data
    go func() {
        for i := 0; i < 1000; i++ {
            ch.In <- fmt.Sprintf("message-%d", i)
        }
    }()
    
    // Receive data
    for msg := range ch.Out {
        fmt.Println("Received:", msg)
    }
    
    // Graceful shutdown
    ch.Stop()      // Stop accepting new messages
    ch.WaitClose() // Wait until all buffered messages are drained
}
```

## ✨ Key Features

| Feature | Description |
|---------|-------------|
| **Generic** | Works with any type: `RevolverChannel16Bit[int]`, `RevolverChannel16Bit[MyStruct]`, etc. |
| **Auto-scaling buffer** | Starts with 1 internal channel; automatically adds more (up to 65,536) when full |
| **FIFO order** | Messages are delivered in the order they were sent |
| **Backpressure** | `ch.In <- val` blocks when all 65,536 slots are full — prevents memory exhaustion |
| **Thread-safe** | Uses `atomic` + `sync.Mutex` for concurrent access |
| **Graceful shutdown** | `Stop()` closes input; `WaitClose()` waits for output drain |
| **Metrics** | `Len()` returns pending items; `Utilization()` shows buffer usage % |


## ⚠️ What NOT to Do

| ❌ Avoid | ✅ Instead |
|----------|-----------|
| Write to `ch.In` after `ch.Stop()` | Check `ch.IsStoped()` first, or use `select` with timeout |
| Assume `WaitClose()` drains `ch.Out` | You must read from `ch.Out` (or drain in a goroutine) before calling `WaitClose()` |
| Use `chCap = 0` | Always set `chCap ≥ 1` for meaningful buffering |
| Ignore backpressure | Handle potential blocking on send with `select` + `default` or context timeout |
| Share `RevolverChannel16Bit` without synchronization | The struct itself is thread-safe, but your payload type `T` may not be |


## Public API Reference

### Constructor
```go
func NewRevolverChannel16Bit[T any](chCap int) *RevolverChannel16Bit[T]
```
Creates a new revolver channel.  
- `chCap`: buffer capacity per internal slot (recommended: 10–1000).  
- Starts worker goroutines automatically.


### Channels
```go
In  chan T  // Send values here
Out chan T  // Receive values here
```


### Control Methods
```go
func (r *RevolverChannel16Bit[T]) Stop()
```
Stops accepting new messages. Closes `In`. Does **not** close `Out` until buffer is drained.

```go
func (r *RevolverChannel16Bit[T]) WaitClose()
```
Blocks until `Out` is closed (i.e., all buffered messages have been read).

```go
func (r *RevolverChannel16Bit[T]) IsStoped() bool
```
Returns `true` if `Stop()` has been called.

```go
func (r *RevolverChannel16Bit[T]) IsClosed() bool
```
Returns `true` if the channel is fully closed (`Out` closed, workers stopped).


### Metrics
```go
func (r *RevolverChannel16Bit[T]) Len() int64
```
Returns the number of messages currently in the buffer (pending delivery).

```go
func (r *RevolverChannel16Bit[T]) Utilization() float64
```
Returns buffer utilization as a percentage (0.00–100.00).  
Formula: `(shiftIn - shiftOut) / 65536 * 100` (with uint16 wraparound support).


## Typical Usage Pattern

```go
ch := NewRevolverChannel16Bit[MyData](100)

// Producer
go func() {
    for item := range source {
        select {
        case ch.In <- item:
            // sent
        case <-ctx.Done():
            return // handle cancellation
        }
    }
}()

// Consumer
go func() {
    for data := range ch.Out {
        process(data)
    }
}()

// Shutdown
ch.Stop()
// Optionally drain remaining items:
go func() { for range ch.Out {} }()
ch.WaitClose()
```

---

## Constants

```go
const limit16bit = 65536  // Maximum number of internal channels
```

The buffer can scale up to `65536 × chCap` pending messages.

---

> **Tip**: For most use cases, `chCap = 100` provides a good balance between memory usage and throughput. Monitor `Utilization()` in production to tune this value.

## Resume

RevolverChannel is ~5 times slower than the native channel due to overhead:

	Managing an array of 65,536 channels
	Mutexes for shiftIn/shiftOut
	Additional buffer expansion logic

However, it offers a unique feature: automatic buffer scaling up to 65,536 × chCap elements without data loss.

> **Bottom line**: If maximum speed is critical, use a native channel. If reliability under peak loads and the ability to survive temporary surges without data loss are more important, RevolverChannel16Bit is an excellent choice.


## Benchmark

go test -test.bench=.* [/revolver_channel]
goos: linux
goarch: amd64
pkg: github.com/claygod/tools/revolver-channel
cpu: Intel(R) Core(TM) i7-6700T CPU @ 2.80GHz

BenchmarkRevolverChannel16_Throughput-8                    	 2798636	       359.1 ns/op	   2785102 ops/sec
BenchmarkRevolverChannel16_Single_Int_Cap1-8               	 1772528	       648.7 ns/op
BenchmarkRevolverChannel16_Single_Int_Cap10-8              	 1719498	       642.9 ns/op
BenchmarkRevolverChannel16_Single_Int_Cap100-8             	 1753002	       634.3 ns/op
BenchmarkRevolverChannel16_Single_String_Cap10-8           	 1663230	       742.6 ns/op
BenchmarkRevolverChannel16_Single_Struct_Cap10-8           	 1818901	       692.1 ns/op
BenchmarkRevolverChannel16_Parallel_8x8_Int_Cap10-8        	 1532665	       793.8 ns/op
BenchmarkRevolverChannel16_Parallel_8x8_Int_Cap100-8       	 1541210	       793.3 ns/op
BenchmarkRevolverChannel16_Parallel_32x32_Int_Cap10-8      	 1522917	       791.5 ns/op
BenchmarkRevolverChannel16_Parallel_32x32_Int_Cap100-8     	 1525963	       784.9 ns/op
BenchmarkRevolverChannel16_Parallel_8x32_Int_Cap10-8       	 1517709	       789.3 ns/op
BenchmarkRevolverChannel16_Parallel_8x32_Int_Cap100-8      	 1498570	       789.6 ns/op
BenchmarkRevolverChannel16_Parallel_32x8_Int_Cap10-8       	 1527642	       787.0 ns/op
BenchmarkRevolverChannel16_Parallel_32x8_Int_Cap100-8      	 1502800	       789.6 ns/op
BenchmarkRevolverChannel16_NativeParallel_Int_Cap10-8      	 1518726	       785.0 ns/op
BenchmarkRevolverChannel16_NativeParallel_Int_Cap100-8     	 1529022	       786.1 ns/op
BenchmarkRevolverChannel16_NativeParallel_String_Cap10-8   	 1494499	       788.7 ns/op
BenchmarkRevolverChannel16_UtilizationUnderLoad-8          	 1528317	       865.8 ns/op	         0.001526 max_utilization_percent
BenchmarkNativeChannel16_Single_Int_Cap10-8                	13179412	        94.36 ns/op
BenchmarkNativeChannel16_Parallel_8x8_Int_Cap10-8          	 6782072	       175.7 ns/op

BenchmarkRevolverChannel8_Throughput-8                     	 3223310	       370.5 ns/op	   2699039 ops/sec
BenchmarkRevolverChannel8_Single_Int_Cap1-8                	 1553918	       683.1 ns/op
BenchmarkRevolverChannel8_Single_Int_Cap10-8               	 1921828	       636.0 ns/op
BenchmarkRevolverChannel8_Single_Int_Cap100-8              	 1631032	       665.5 ns/op
BenchmarkRevolverChannel8_Single_String_Cap10-8            	 1785456	       650.1 ns/op
BenchmarkRevolverChannel8_Single_Struct_Cap10-8            	 1794909	       638.1 ns/op
BenchmarkRevolverChannel8_Parallel_8x8_Int_Cap10-8         	 1509327	       790.6 ns/op
BenchmarkRevolverChannel8_Parallel_8x8_Int_Cap100-8        	 1514036	       790.2 ns/op
BenchmarkRevolverChannel8_Parallel_32x32_Int_Cap10-8       	 1520601	       813.4 ns/op
BenchmarkRevolverChannel8_Parallel_32x32_Int_Cap100-8      	 1516646	       784.9 ns/op
BenchmarkRevolverChannel8_Parallel_8x32_Int_Cap10-8        	 1522470	       784.9 ns/op
BenchmarkRevolverChannel8_Parallel_8x32_Int_Cap100-8       	 1499706	       796.3 ns/op
BenchmarkRevolverChannel8_Parallel_32x8_Int_Cap10-8        	 1521294	       788.7 ns/op
BenchmarkRevolverChannel8_Parallel_32x8_Int_Cap100-8       	 1510764	       797.6 ns/op
BenchmarkRevolverChannel8_NativeParallel_Int_Cap10-8       	 1506211	       786.6 ns/op
BenchmarkRevolverChannel8_NativeParallel_Int_Cap100-8      	 1541803	       785.9 ns/op
BenchmarkRevolverChannel8_NativeParallel_String_Cap10-8    	 1375261	       793.1 ns/op
BenchmarkRevolverChannel8_UtilizationUnderLoad-8           	 1504680	       781.9 ns/op	         0.3906 max_utilization_percent
BenchmarkNativeChannel8_Single_Int_Cap10-8                 	13808708	        85.65 ns/op
BenchmarkNativeChannel8_Parallel_8x8_Int_Cap10-8           	 6847455	       174.4 ns/op

Parallel Benchmark Stability: All configurations (8x8, 32x32, 8x32, 32x8) show nearly identical results (~780-804 ns/op). 
This means the implementation scales predictably without degradation as the number of goroutines increases.
Low overhead for single-thread execution: ~630-690 ns/op—acceptable for autoscaling functionality.
Data type has little impact: int, string, and struct show similar results, indicating that generics are efficient.

### Copyright

Copyright © 2026 Eduard Sesigin. All rights reserved. Contacts: <claygod@yandex.ru>