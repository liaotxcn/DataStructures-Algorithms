package主干

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// Config 雪花算法配置结构
type Config struct {
	Epoch          int64 // 起始时间戳(毫秒)，通常是系统上线时间
	WorkerIDBits   uint8 // 机器ID位数，决定最多支持多少台机器
	DataCenterBits uint8 // 数据中心位数，决定最多支持多少个数据中心
	SequenceBits   uint8 // 序列号位数，决定每毫秒能生成多少个ID
	MaxTimeDrift   int64 // 最大允许时钟回拨(毫秒)，超过此值会报错
}

// DefaultConfig 默认配置
var DefaultConfig = Config{
	Epoch:          1609459200000, // 2025-01-01 00:00:00 UTC
	WorkerIDBits:   5,             // 32个worker
	DataCenterBits: 5,             // 32个数据中心
	SequenceBits:   12,            // 每毫秒4096个ID
	MaxTimeDrift:   10,            // 允许10毫秒时钟回拨
}

// IDComponents 解析出的ID组成部分
type IDComponents struct {
	Timestamp    int64 // 时间戳部分
	DataCenterID int64 // 数据中心ID部分
	WorkerID     int64 // 机器ID部分
	Sequence     int64 // 序列号部分
}

// Snowflake 优化的雪花算法结构体
type Snowflake struct {
	mu              sync.Mutex    // 互斥锁，保证并发安全
	config          Config        // 配置信息
	lastTimestamp   int64         // 上次生成ID的时间戳
	workerID        int64         // 机器ID
	dataCenterID    int64         // 数据中心ID
	sequence        int64         // 当前序列号
	timeDriftCount  int32         // 时钟回拨次数统计(原子操作)
	idPool          chan int64    // ID缓冲池
	stopChan        chan struct{} // 停止信号通道
	poolSize        int           // 池大小
	maxWorkerID     int64         // 最大workerID值
	maxDataCenter   int64         // 最大数据中心ID值
	maxSequence     int64         // 最大序列号值
	timeShift       uint8         // 时间戳左移位数
	dataCenterShift uint8         // 数据中心ID左移位数
	workerShift     uint8         // 机器ID左移位数
}

// NewWithConfig 使用自定义配置创建Snowflake实例
func NewWithConfig(workerID, dataCenterID int64, config Config) (*Snowflake, error) {
	// 计算各部分的掩码和最大值
	// 使用位运算计算最大值：-1左移n位后取反
	maxWorkerID := int64(-1) ^ (int64(-1) << config.WorkerIDBits)
	maxDataCenter := int64(-1) ^ (int64(-1) << config.DataCenterBits)
	maxSequence := int64(-1) ^ (int64(-1) << config.SequenceBits)

	// 检查workerID和数据中心ID是否在有效范围内
	if workerID < 0 || workerID > maxWorkerID {
		return nil, fmt.Errorf("worker ID must be between 0 and %d", maxWorkerID)
	}
	if dataCenterID < 0 || dataCenterID > maxDataCenter {
		return nil, fmt.Errorf("data center ID must be between 0 and %d", maxDataCenter)
	}

	// 计算各部分位移量
	timeShift := config.WorkerIDBits + config.DataCenterBits + config.SequenceBits
	dataCenterShift := config.WorkerIDBits + config.SequenceBits
	workerShift := config.SequenceBits

	sf := &Snowflake{
		config:          config,
		workerID:        workerID,
		dataCenterID:    dataCenterID,
		maxWorkerID:     maxWorkerID,
		maxDataCenter:   maxDataCenter,
		maxSequence:     maxSequence,
		timeShift:       timeShift,
		dataCenterShift: dataCenterShift,
		workerShift:     workerShift,
	}

	return sf, nil
}

// New 使用默认配置创建Snowflake实例
func New(workerID, dataCenterID int64) (*Snowflake, error) {
	return NewWithConfig(workerID, dataCenterID, DefaultConfig)
}

// NewWithPool 创建带ID池的Snowflake实例
func NewWithPool(workerID, dataCenterID int64, poolSize int) (*Snowflake, error) {
	// 先创建基础实例
	sf, err := New(workerID, dataCenterID)
	if err != nil {
		return nil, err
	}

	// 初始化池相关字段
	sf.poolSize = poolSize
	sf.idPool = make(chan int64, poolSize)
	sf.stopChan = make(chan struct{})

	// 启动后台goroutine填充ID池
	go sf.fillIDPool()

	return sf, nil
}

// fillIDPool 填充ID池的后台任务
func (s *Snowflake) fillIDPool() {
	for {
		select {
		case <-s.stopChan: // 收到停止信号
			return
		default:
			// 如果池中ID少于一半容量，开始填充
			if len(s.idPool) < s.poolSize/2 {
				id, err := s.generateID()
				if err != nil {
					// 生成失败时短暂等待后重试
					time.Sleep(10 * time.Millisecond)
					continue
				}
				select {
				case.idPool <- id: // 将生成的ID放入池中
				default:
					// 池已满，短暂等待
					time.Sleep(100 * time.Microsecond)
				}
			} else {
				// 池足够满，短暂休眠避免CPU空转
				time.Sleep(10 * time.Millisecond)
			}
		}
	}
}

// generateID 生成单个ID的核心方法
func (s *Snowflake) generateID() (int64, error) {
.mu.Lock()         // 加锁保证并发安全
	defer.mu.Unlock() // 方法结束时解锁

	now := currentMillis() // 获取当前时间戳(毫秒)

	// 处理时钟回拨情况
	if now < s.lastTimestamp {
		drift := s.lastTimestamp - now // 计算回拨的时间差
		// 如果回拨在允许范围内
		if drift <= s.config.MaxTimeDrift {
			atomic.AddInt32(&s.timeDriftCount, 1) // 增加回拨计数
			// 等待直到时间追上上次记录的时间
			for now < s.lastTimestamp {
				time.Sleep(time.Millisecond)
				now = currentMillis()
			}
		} else {
			// 回拨超过最大允许值，返回错误
			return 0, ErrClockBackwards
		}
	}

	// 同一毫秒内生成序列号
	if now == s.lastTimestamp {
		// 序列号自增并与最大值取模
.sequence = (s.sequence + 1) & s.maxSequence
		// 如果序列号归零，表示当前毫秒内ID已用完，等待下一毫秒
		if.sequence == 0 {
			now = s.waitNextMillis(now)
		}
	} else {
		// 新的一毫秒，序列号从0开始
.sequence = 0
	}

	// 更新最后时间戳
.lastTimestamp = now

	// 组合各部分生成最终ID
	id := ((now - s.config.Epoch) << s.timeShift) | // 时间戳部分左移
		(s.dataCenterID << s.dataCenterShift) | // 数据中心部分左移
		(s.workerID << s.workerShift) | // worker部分左移
.sequence // 序列号部分

	return id, nil
}

// NextID 获取下一个ID（带池化版本）
func (s *Snowflake) NextID() (int64, error) {
	// 如果启用了池化
	if.idPool != nil {
		select {
		case id := <-s.idPool: // 从池中获取ID
			return id, nil
		case <-time.After(50 * time.Millisecond):
			// 池为空，超时后直接生成
			return.generateID()
		}
	}
	// 未启用池化，直接生成
	return.generateID()
}

// ParseID 解析ID为各组成部分
func (s *Snowflake) ParseID(id int64) IDComponents {
	// 通过右移和掩码提取各部分值
	timestamp := (id >> s.timeShift) + s.config.Epoch
	dataCenterID := (id >> s.dataCenterShift) & s.maxDataCenter
	workerID := (id >> s.workerShift) & s.maxWorkerID
	sequence := id & s.maxSequence

	return IDComponents{
		Timestamp:    timestamp,
		DataCenterID: dataCenterID,
		WorkerID:     workerID,
		Sequence:     sequence,
	}
}

// TimeFromID 从ID中提取时间
func (s *Snowflake) TimeFromID(id int64) time.Time {
	// 提取时间戳部分并转换为time.Time
	timestamp := (id >> s.timeShift) + s.config.Epoch
	return time.Unix(timestamp/1000, (timestamp%1000)*1e6)
}

// Close 关闭ID池
func (s *Snowflake) Close() {
	if.stopChan != nil {
		close(s.stopChan) // 关闭通道，通知后台goroutine退出
	}
}

// waitNextMillis 等待直到下一毫秒
func (s *Snowflake) waitNextMillis(last int64) int64 {
	now := currentMillis()
	// 循环等待直到进入下一毫秒
	for now <= last {
		time.Sleep(100 * time.Microsecond) // 短暂休眠
		now = currentMillis()
	}
	return now
}

// GetTimeDriftCount 获取时钟回拨次数
func (s *Snowflake) GetTimeDriftCount() int32 {
	return atomic.LoadInt32(&s.timeDriftCount) // 原子读取计数器
}

// currentMillis 获取当前毫秒数
func currentMillis() int64 {
	return time.Now().UnixNano() / 1e6 // 纳秒转毫秒
}

// 定义错误变量
var (
	ErrClockBackwards = errors.New("clock moved backwards")
)

func main() {
	// 1. 创建带池的雪花算法实例
	sf, err := NewWithPool(1, 1, 1000)
	if err != nil {
		log.Fatalf("Failed to create snowflake: %v", err)
	}
	defer sf.Close()

	// 2. 并发测试
	var wg sync.WaitGroup
	start := time.Now()
	const goroutines = 100       // 并发goroutine数量
	const idsPerGoroutine = 1000 // 每个goroutine生成的ID数量

	// 启动多个goroutine并发生成ID
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < idsPerGoroutine; j++ {
				id, err := sf.NextID()
				if err != nil {
					log.Printf("Error generating ID: %v", err)
					continue
				}
				// 解析ID并丢弃结果(仅演示)
				_ = sf.ParseID(id)
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)
	fmt.Printf("Generated %d IDs in %v (%d IDs/sec)\n",
		goroutines*idsPerGoroutine, duration,
		int64(float64(goroutines*idsPerGoroutine)/duration.Seconds()))

	// 3. 解析示例
	sampleID, _ := sf.NextID()
	fmt.Printf("\nSample ID: %d\n", sampleID)
	components := sf.ParseID(sampleID)
	fmt.Printf("Parsed: Timestamp=%d, DataCenter=%d, Worker=%d, Sequence=%d\n",
		components.Timestamp, components.DataCenterID, components.WorkerID, components.Sequence)
	fmt.Println("Created at:", sf.TimeFromID(sampleID).Format("2006-01-02 15:04:05.000"))

	// 4. 监控信息
	fmt.Printf("\nTime drift events: %d\n", sf.GetTimeDriftCount())
}
