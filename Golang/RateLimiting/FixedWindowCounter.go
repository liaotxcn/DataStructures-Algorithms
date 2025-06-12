package主干

import (
	"log"
	"sync/atomic"
	"time"
)

// FixedWindowLimiter 固定窗口限流器结构体
type FixedWindowLimiter struct {
	windowSize  int64 // 窗口大小（毫秒）
	maxRequests int32 // 窗口内最大请求数
	counter     int32 // 当前窗口内的请求数
	lastWindow  int64 // 上一个窗口的开始时间戳（毫秒）
}

// NewFixedWindowLimiter 创建一个新的固定窗口限流器
func NewFixedWindowLimiter(windowSize int64, maxRequests int32) *FixedWindowLimiter {
	return &FixedWindowLimiter{
		windowSize:  windowSize,
		maxRequests: maxRequests,
		counter:     0,
		lastWindow:  time.Now().UnixNano() / int64(time.Millisecond),
	}
}

// Allow 判断是否允许请求通过
func (f *FixedWindowLimiter) Allow() bool {
	now := time.Now().UnixNano() / int64(time.Millisecond)
	lastWindow := atomic.LoadInt64(&f.lastWindow)

	if now-lastWindow > f.windowSize {
		// 新窗口开始，重置计数器
		if atomic.CompareAndSwapInt64(&f.lastWindow, lastWindow, now) {
			atomic.StoreInt32(&f.counter, 0)
		}
	}

	for {
		count := atomic.LoadInt32(&f.counter)
		if count >= f.maxRequests {
			log.Println("请求被限流")
			return false
		}
		if atomic.CompareAndSwapInt32(&f.counter, count, count+1) {
			log.Println("请求通过")
			return true
		}
	}
}

func main() {
	// 这里可以添加测试限流逻辑的代码
	limiter := NewFixedWindowLimiter(1000, 10) // 创建一个每秒最多 10 个请求的限流器
	if limiter.Allow() {
		println("请求通过")
	} else {
		println("请求被限流")
	}
}
