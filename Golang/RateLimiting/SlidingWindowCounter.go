package主干

import (
	"fmt"
	"time"
)

var LimitQueue = make(map[string][]int64)

// 单机时间滑动窗口限流法
func LimitFreqSingle(queueName string, count uint, timeWindow int64) bool {
	currTime := time.Now().Unix()

	if _, ok := LimitQueue[queueName]; !ok {
		LimitQueue[queueName] = make([]int64, 0)
	}

	// 队列未满
	if uint(len(LimitQueue[queueName])) < count {
		LimitQueue[queueName] = append(LimitQueue[queueName], currTime)
		return true
	}

	// 队列满了，取出最早访问的时间
	earlyTime := LimitQueue[queueName][0]

	// 说明最早期的时间还在时间窗口内，还没过期，所以不允许通过
	if currTime-earlyTime <= timeWindow {
		return false
	} else {
		// 说明最早期的访问应该过期了，去掉最早期的
		LimitQueue[queueName] = LimitQueue[queueName][1:]
		LimitQueue[queueName] = append(LimitQueue[queueName], currTime)
	}

	return true
}

func main() {
	if LimitFreqSingle("api_request", 5, 10) {
		fmt.Println("请求通过")
	} else {
		fmt.Println("请求被限流")
	}
}
