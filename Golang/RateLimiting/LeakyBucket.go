package主干

import (
	"fmt"
	"sync"
	"time"
)

// LeakyBucket 定义漏桶结构体，用于实现漏桶算法限流。
type LeakyBucket struct {
	capacity     int64      // 漏桶的最大容量，即最多能容纳的请求数量。
	rate         int64      // 漏桶漏水的速率，单位为每秒漏出的请求数量。
	water        int64      // 当前漏桶中的水量，即已接收但还未处理的请求数量。
	lastLeakTime time.Time  // 上次漏水的时间戳，用于计算本次需要漏出的水量。
	mutex        sync.Mutex // 互斥锁，保证在并发环境下对漏桶状态的操作是线程安全的。
}

// NewLeakyBucket 创建一个新的漏桶实例。
// capacity 是漏桶的最大容量，rate 是漏水的速率。
func NewLeakyBucket(capacity, rate int64) *LeakyBucket {
	return &LeakyBucket{
		capacity:     capacity,
		rate:         rate,
		water:        0,
		lastLeakTime: time.Now(),
	}
}

// Allow 检查当前请求是否可以通过漏桶。
// 如果漏桶有足够的空间容纳新请求，则返回 true；否则返回 false。
func (lb *LeakyBucket) Allow() bool {
	lb.mutex.Lock()         // 加锁，保证并发安全
	defer lb.mutex.Unlock() // 函数结束时解锁

	// 计算从上次漏水到现在经过的时间
	now := time.Now()
	elapsed := now.Sub(lb.lastLeakTime).Seconds()

	// 计算这段时间内应该漏出的水量
	leakedWater := int64(elapsed) * lb.rate

	// 更新漏桶中的水量，确保水量不会小于 0
	if leakedWater > lb.water {
		lb.water = 0
	} else {
		lb.water -= leakedWater
	}

	// 更新上次漏水的时间戳
	lb.lastLeakTime = now

	// 检查是否有足够的空间容纳新请求
	if lb.water+1 <= lb.capacity {
		lb.water++ // 有空间，接收新请求，水量加 1
		return true
	}

	return false // 没有空间，请求被限流
}

func main() {
	// 创建一个容量为 10，速率为每秒 2 个请求的漏桶
	bucket := NewLeakyBucket(10, 2)

	// 模拟 20 个请求
	for i := 0; i < 20; i++ {
		if bucket.Allow() {
			fmt.Printf("请求 %d 通过\n", i+1)
		} else {
			fmt.Printf("请求 %d 被限流\n", i+1)
		}
		time.Sleep(200 * time.Millisecond)
	}
}
