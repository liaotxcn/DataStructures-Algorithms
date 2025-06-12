package主干

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// Redis实现滑动窗口限流
// 使用有序集合（ZSet）来管理滑动窗口内的请求
// 每个请求在ZSet中作为一个成员，分数为请求的时间戳
var rdb = redis.NewClient(&redis.Options{
	Addr:     "localhost:6379", // Redis Server
	Password: "",               // 无密码
	DB:       0,                // 使用默认数据库
})

// 窗口大小，单位为秒
const windowSize = 60

// 窗口内允许的最大请求数
const limit = 100

// slidingWindowRateLimit 实现滑动窗口限流逻辑
// key 是用于标识限流的键，不同的API可以使用不同的键
func slidingWindowRateLimit(key string) bool {
	// 创建一个上下文，用于控制Redis操作的生命周期
	ctx := context.Background()
	// 获取当前时间的时间戳（秒）
	currentTime := time.Now().Unix()
	// 计算窗口的起始时间
	windowStart := currentTime - windowSize

	// 清理过期请求，删除ZSet中分数小于窗口起始时间的成员
	_, err := rdb.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart)).Result()
	if err != nil {
		// 打印错误信息，方便调试
		fmt.Println("Error removing expired requests:", err)
		return false
	}

	// 累加新请求，将当前时间戳作为成员添加到ZSet中，分数为当前时间戳
	_, err = rdb.ZAdd(ctx, key, &redis.Z{Score: float64(currentTime), Member: currentTime}).Result()
	if err != nil {
		// 打印错误信息，方便调试
		fmt.Println("Error adding new request:", err)
		return false
	}

	// 统计窗口内的请求数，计算ZSet中分数在窗口起始时间和当前时间之间的成员数量
	count, err := rdb.ZCount(ctx, key, fmt.Sprintf("%d", windowStart), fmt.Sprintf("%d", currentTime)).Result()
	if err != nil {
		// 打印错误信息，方便调试
		fmt.Println("Error counting requests:", err)
		return false
	}

	// 判断是否超过限制，如果请求数超过允许的最大请求数，则返回false，表示请求被限流
	if count > int64(limit) {
		return false
	}

	return true
}

func main() {
	if slidingWindowRateLimit("api_request") {
		fmt.Println("请求通过")
	} else {
		fmt.Println("请求被限流")
	}
}
