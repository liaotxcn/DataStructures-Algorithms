package主干

import (
	"fmt"
	"sync/atomic"
	"time"
)

// TokenBucket 表示令牌桶结构体
// 该结构体用于管理令牌桶的状态，包括容量、令牌生成速率、当前可用令牌数等信息
// 使用原子操作保证并发安全
type TokenBucket struct {
	// capacity 表示令牌桶的最大容量，即令牌桶最多能容纳的令牌数量
	capacity int64
	// rate 表示令牌生成速率，即每秒生成的令牌数
	rate int64
	// available 表示当前可用的令牌数，使用原子操作保证并发安全
	available int64
	// lastRefill 表示上次填充令牌的时间戳，单位为纳秒，用于计算新生成的令牌数
	lastRefill int64
}

// NewTokenBucket 创建一个新的令牌桶
// 参数 capacity 为令牌桶的最大容量，rate 为令牌生成速率（每秒生成的令牌数）
// 返回一个指向新创建的令牌桶的指针
func NewTokenBucket(capacity, rate int64) *TokenBucket {
	// 获取当前时间的纳秒时间戳
	now := time.Now().UnixNano()
	return &TokenBucket{
		// 设置令牌桶的最大容量
		capacity: capacity,
		// 设置令牌生成速率
		rate: rate,
		// 初始化时，令牌桶为满状态
		available: capacity,
		// 记录上次填充令牌的时间戳
		lastRefill: now,
	}
}

// Allow 检查是否允许请求通过
// 该方法会根据当前令牌桶的状态，判断是否有足够的令牌来处理请求
// 如果有足够的令牌，会消耗一个令牌并返回 true；否则返回 false
func (tb *TokenBucket) Allow() bool {
	// 获取当前时间的纳秒时间戳
	now := time.Now().UnixNano()
	// 原子性地加载上次填充令牌的时间戳
	lastRefill := atomic.LoadInt64(&tb.lastRefill)
	// 计算从上次填充令牌到现在经过的秒数
	elapsed := (now - lastRefill) / int64(time.Second)
	// 计算这段时间内新生成的令牌数
	newTokens := elapsed * tb.rate

	// 如果有新生成的令牌
	if newTokens > 0 {
		// 原子性地增加可用令牌数
		newAvailable := atomic.AddInt64(&tb.available, newTokens)
		// 如果可用令牌数超过了令牌桶的最大容量
		if newAvailable > tb.capacity {
			// 将可用令牌数设置为令牌桶的最大容量
			newAvailable = tb.capacity
			// 原子性地存储新的可用令牌数
			atomic.StoreInt64(&tb.available, newAvailable)
		}
		// 原子性地更新上次填充令牌的时间戳为当前时间
		atomic.StoreInt64(&tb.lastRefill, now)
	}

	// 尝试原子性地消耗一个令牌
	// 如果消耗后可用令牌数仍然大于等于 0，则表示请求可以通过
	return atomic.AddInt64(&tb.available, -1) >= 0
}

func main() {
	// 创建一个容量为 10，每秒生成 5 个令牌的令牌桶
	limiter := NewTokenBucket(10, 5)

	// 模拟 20 次请求
	for i := 0; i < 20; i++ {
		if limiter.Allow() {
			fmt.Printf("请求 %d 通过\n", i+1)
		} else {
			fmt.Printf("请求 %d 被限流\n", i+1)
		}
		// 每次请求间隔 100 毫秒
		time.Sleep(100 * time.Millisecond)
	}
}
