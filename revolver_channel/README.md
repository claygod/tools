# Revolver Channel

A generic, auto-scaling buffered channel for Go.

A high-performance, concurrent-safe rotating channel implementation for Go. `RevolverChannel` uses a circular buffer of channels coordinated by `CircleSectorControl`, enabling efficient multi-producer/multi-consumer scenarios with dynamic capacity scaling.

Unlike standard Go channels, `RevolverChannel` automatically rotates through multiple internal channels when capacity is reached, preventing blocking while maintaining order and providing visibility into buffer utilization.


## 📦 Installation

```bash
go get github.com/claygod/tools/revolver_channel
```


## 🔌 API Reference

### Constructor

#### `NewRevolverChannel[T any](chCap int, chCount int) (*RevolverChannel[T], error)`

Creates a new `RevolverChannel` instance.

**Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `chCap` | `int` | Capacity of **each individual internal channel** (buffer size per sector) |
| `chCount` | `int` | **Number of internal channels** in the rotation pool (total sectors) |

**Returns:**
- `*RevolverChannel[T]` — Pointer to the new instance
- `error` — Error if parameters are invalid

**Errors:**
- `chCap <= 0` — Returns error
- `chCount <= 0` — Returns error

**Example:**
```go
rCh, err := revolver_channel.NewRevolverChannel[int](100, 65536)
if err != nil {
    log.Fatalf("Failed to create channel: %v", err)
}
defer cleanupChannel(rCh)
```


### Public Methods

| Method | Description | Returns |
|--------|-------------|---------|
| `In` (field) | Send values into the channel | `chan T` |
| `Out` (field) | Receive values from the channel | `chan T` |
| `Stop()` | Gracefully stop the channel | `void` |
| `WaitClose()` | Block until channel is fully closed | `void` |
| `IsStoped()` | Check if stop was initiated | `bool` |
| `IsClosed()` | Check if channel is fully closed | `bool` |
| `Len()` | Get current number of items in buffer | `int64` |
| `Utilization()` | Get buffer utilization percentage (0–100%) | `float64` |


## ⚙️ Configuration Guide: `chCap` and `chCount`

The performance and memory characteristics of `RevolverChannel` depend heavily on proper configuration of `chCap` and `chCount`.

### 📊 Parameter Overview

| Parameter | Controls | Memory Impact | Performance Impact |
|-----------|----------|---------------|-------------------|
| `chCap` | Buffer size **per sector** | `chCap × chCount × sizeof(T)` | Higher = less rotation, more memory per sector |
| `chCount` | Number of **sectors** in rotation | Linear | Higher = more headroom for bursts, more atomic ops |


## 📏 Recommended Values

### Minimum Values

| Parameter | Minimum | When to Use |
|-----------|---------|-------------|
| `chCap` | `1` | Testing, minimal memory footprint |
| `chCount` | `2` | Absolute minimum (1 active + 1 buffer) |

**⚠️ Warning:** `chCount = 1` is **not recommended** — it defeats the purpose of rotation and will block immediately when the single channel fills.

```go
// Minimum viable configuration
rCh, _ := NewRevolverChannel[int](1, 2)
```


### Maximum Recommended Values

| Parameter | Maximum | Reason |
|-----------|---------|--------|
| `chCap` | `10,000` | Beyond this, consider a single large channel |
| `chCount` | `262,144` (2^18) | Memory and atomic operation overhead |

**Memory Calculation:**
```
Memory ≈ chCap × chCount × sizeof(T)

Example: chCap=100, chCount=65536, T=int64
Memory ≈ 100 × 65536 × 8 bytes = 52.4 MB
```


### Optimal Configurations by Use Case

| Use Case | chCap | chCount | Rationale |
|----------|-------|---------|-----------|
| **Low latency, low throughput** | `10–50` | `1,024–4,096` | Minimal memory, fast rotation |
| **General purpose** | `100` | `65,536` | Balanced (default recommendation) |
| **High throughput, bursty** | `100–500` | `131,072–262,144` | Headroom for writer spikes |
| **Memory constrained** | `1–10` | `256–1,024` | Minimal footprint |
| **Large payloads (structs)** | `10–50` | `4,096–16,384` | Reduce memory per sector |
| **Small payloads (int, bool)** | `100–1,000` | `65,536–131,072` | Maximize throughput |


## 📈 Handling Load Spikes (Writer Overtaking Reader)

The key advantage of `RevolverChannel` is handling temporary imbalances where the writer produces faster than the reader consumes.

### Utilization Thresholds

| Utilization | Status | Action |
|-------------|--------|--------|
| `0–10%` | Normal | No action needed |
| `10–50%` | Moderate | Monitor |
| `50–80%` | High | Consider increasing `chCount` |
| `80–100%` | Critical | Writer will block; increase `chCount` or `chCap` |

### Calculating Required `chCount`

```go
// Formula for expected burst capacity
requiredChCount := (burstSize / chCap) + safetyMargin

// Example: Expect 100,000 items burst, chCap=100
requiredChCount := (100000 / 100) + 10  // = 1,010 sectors
```

### Recommended Safety Margins

| Scenario | Safety Margin |
|----------|---------------|
| Predictable load | `+10%` |
| Variable load | `+50%` |
| Unpredictable bursts | `+100%` |


## 📚 Usage Examples

### Example 1: Basic Usage

```go
package main

import (
    "fmt"
    "log"
    revolver_channel "github.com/claygod/tools/revolver_channel"
)

func main() {
    rCh, err := revolver_channel.NewRevolverChannel[int](100, 65536)
    if err != nil {
        log.Fatalf("Failed to create channel: %v", err)
    }

    // Send values
    go func() {
        for i := 0; i < 1000; i++ {
            rCh.In <- i
        }
        rCh.Stop()
    }()

    // Receive values
    go func() {
        for val := range rCh.Out {
            fmt.Println("Received:", val)
        }
    }()

    // Wait for completion
    rCh.WaitClose()
    fmt.Println("Channel closed")
}
```


### Example 2: Monitoring Utilization

```go
package main

import (
    "fmt"
    "time"
    revolver_channel "github.com/claygod/tools/revolver_channel"
)

func main() {
    rCh, _ := revolver_channel.NewRevolverChannel[int](100, 65536)

    // Monitor utilization
    go func() {
        ticker := time.NewTicker(100 * time.Millisecond)
        defer ticker.Stop()
        for range ticker.C {
            util := rCh.Utilization()
            fmt.Printf("Utilization: %.2f%%\n", util)
            
            if util > 80 {
                fmt.Println("⚠️  Warning: High utilization!")
            }
        }
    }()

    // Simulate burst
    for i := 0; i < 10000; i++ {
        rCh.In <- i
    }

    rCh.Stop()
    go func() { for range rCh.Out {} }()
    rCh.WaitClose()
}
```


### Example 3: Configuration for Different Scenarios

```go
// Scenario 1: IoT sensor data (small, frequent, predictable)
iotChannel, _ := NewRevolverChannel[SensorData](50, 4096)

// Scenario 2: Web request processing (variable, bursty)
webChannel, _ := NewRevolverChannel[Request](200, 131072)

// Scenario 3: Log aggregation (high volume, large payloads)
logChannel, _ := NewRevolverChannel[LogEntry](10, 262144)

// Scenario 4: Testing/minimal footprint
testChannel, _ := NewRevolverChannel[int](1, 256)
```


## 🔍 Performance Characteristics

### Benchmark Results (Intel i7-6700T)

| Benchmark | Throughput | Latency |
|-----------|------------|---------|
| Single-threaded | ~2.9M ops/sec | ~650 ns/op |
| Parallel 8×8 | ~2.5M ops/sec | ~780 ns/op |
| Parallel 32×32 | ~2.5M ops/sec | ~780 ns/op |
| Native channel (baseline) | ~12M ops/sec | ~84 ns/op |

**Overhead:** ~4–8× compared to native channel (expected for added functionality)


### Memory Footprint

```
Base overhead: ~1 KB per RevolverChannel instance
Per sector:    sizeof(chan T) ≈ 16 bytes (pointer)
Per item:      sizeof(T) in channel buffer

Total ≈ 1KB + (chCount × 16) + (chCount × chCap × sizeof(T))

Example: chCap=100, chCount=65536, T=int64
Total ≈ 1KB + 1MB + 52MB = ~53MB
```


## ⚠️ Important Notes

| Aspect | Description |
|--------|-------------|
| **Thread-Safety** | All operations are safe for concurrent use |
| **Ordering** | FIFO order is preserved within each sector; global order may vary under high concurrency |
| **Blocking** | Writer blocks only when all sectors are full (utilization = 100%) |
| **Memory** | Channels are allocated on-demand; nil sectors consume minimal memory |
| **Graceful Shutdown** | Always call `Stop()` then `WaitClose()` to avoid goroutine leaks |
| **Drain Out** | After `Stop()`, drain `rCh.Out` in a goroutine to allow `workerOut` to complete |


## 🛠️ Helper: Safe Cleanup

```go
func cleanupChannel[T any](rCh *RevolverChannel[T]) {
    rCh.Stop()
    go func() {
        for range rCh.Out {
            // Drain
        }
    }()
    rCh.WaitClose()
}
```

**Usage:**
```go
rCh, _ := NewRevolverChannel[int](100, 65536)
defer cleanupChannel(rCh)
```


## 🧪 Running Tests

```bash
# All tests
go test -v ./...

# With race detector
go test -race -v ./...

# Benchmarks
go test -bench=. -benchmem ./...

# Specific benchmark
go test -bench=BenchmarkRevolverChannel_Single_Int_Cap10 -benchmem ./...

# Short mode (skip long tests)
go test -short -v ./...
```

## Benchmark

go test -test.bench=.* [/revolver_channel]
goos: linux
goarch: amd64
pkg: github.com/claygod/tools/revolver-channel
cpu: Intel(R) Core(TM) i7-6700T CPU @ 2.80GHz

BenchmarkRevolverChannel_Throughput-8                      	 3215754	       346.6 ns/op	   2885307 ops/sec
BenchmarkRevolverChannel_Single_Int_Cap1-8                 	 1656493	       726.5 ns/op
BenchmarkRevolverChannel_Single_Int_Cap10-8                	 1922991	       649.7 ns/op
BenchmarkRevolverChannel_Single_Int_Cap100-8               	 1602944	       659.0 ns/op
BenchmarkRevolverChannel_Single_String_Cap10-8             	 1960214	       693.8 ns/op
BenchmarkRevolverChannel_Single_Struct_Cap10-8             	 1909804	       608.8 ns/op
BenchmarkRevolverChannel_Parallel_8x8_Int_Cap10-8          	 1531141	       779.0 ns/op
BenchmarkRevolverChannel_Parallel_8x8_Int_Cap100-8         	 1540791	       778.2 ns/op
BenchmarkRevolverChannel_Parallel_32x32_Int_Cap10-8        	 1538475	       782.1 ns/op
BenchmarkRevolverChannel_Parallel_32x32_Int_Cap100-8       	 1548038	       779.9 ns/op
BenchmarkRevolverChannel_Parallel_8x32_Int_Cap10-8         	 1527771	       778.0 ns/op
BenchmarkRevolverChannel_Parallel_8x32_Int_Cap100-8        	 1528088	       778.2 ns/op
BenchmarkRevolverChannel_Parallel_32x8_Int_Cap10-8         	 1531784	       779.3 ns/op
BenchmarkRevolverChannel_Parallel_32x8_Int_Cap100-8        	 1540744	       780.9 ns/op
BenchmarkRevolverChannel_NativeParallel_Int_Cap10-8        	 1543104	       776.6 ns/op
BenchmarkRevolverChannel_NativeParallel_Int_Cap100-8       	 1548196	       782.9 ns/op
BenchmarkRevolverChannel_NativeParallel_String_Cap10-8     	 1536213	       782.6 ns/op
BenchmarkRevolverChannel_UtilizationUnderLoad-8            	 1678024	       736.4 ns/op	         0.001526 max_utilization_percent
BenchmarkNativeChannel_Single_Int_Cap10-8                  	13413020	        84.31 ns/op
BenchmarkNativeChannel_Parallel_8x8_Int_Cap10-8            	 6743008	       175.6 ns/op
BenchmarkRevolverChannel_Small_ChCount_100-8               	 1868284	       678.2 ns/op
BenchmarkRevolverChannel_Small_ChCount_1000-8              	 1827271	       657.5 ns/op
BenchmarkRevolverChannel_Utilization_Low-8                 	166953406	         6.079 ns/op	         0.001526 avg_utilization_percent
BenchmarkRevolverChannel_Utilization_High-8                	 1659066	       741.6 ns/op	         1.000 max_utilization_percent

Parallel Benchmark Stability: All configurations (8x8, 32x32, 8x32, 32x8) show nearly identical results (~780 ns/op). 
This means the implementation scales predictably without degradation as the number of goroutines increases.
Low overhead for single-thread execution: ~630-690 ns/op—acceptable for autoscaling functionality.
Data type has little impact: int, string, and struct show similar results, indicating that generics are efficient.

## 📄 License

Copyright © 2026 Eduard Sesigin. All rights reserved.

Contact: [claygod@yandex.ru](mailto:claygod@yandex.ru)


## 🎯 Quick Reference Card

| Decision | Recommendation |
|----------|---------------|
| **Starting point** | `chCap=100, chCount=65536` |
| **Memory constrained** | `chCap=10, chCount=1024` |
| **High burst tolerance** | `chCap=100, chCount=131072` |
| **Large payloads** | `chCap=10–50, chCount=4096–16384` |
| **Monitor threshold** | Alert if `Utilization() > 80%` |
| **Never use** | `chCount < 2` or `chCap < 1` |


**Happy coding!** 🚀
