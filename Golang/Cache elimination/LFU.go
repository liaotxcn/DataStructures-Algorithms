package main

import (
	"container/heap"
	"fmt"
	"sync"
	"testing"
	"time"
)

// LFUCache 线程安全的LFU缓存结构
// 使用哈希表+最小堆实现，哈希表提供O(1)访问，最小堆维护使用频率
type LFUCache struct {
	capacity int                    // 缓存容量
	cache    map[interface{}]*entry // 哈希表存储键和entry指针
	heap     *minHeap               // 最小堆，按使用频率排序
	lock     sync.RWMutex           // 读写锁保证线程安全
	stats    struct {               // 运行时统计信息
		hits         int64 // 命中次数
		misses       int64 // 未命中次数
		evictions    int64 // 淘汰次数
		expiredCount int64 // 过期条目数
	}
	stopChan chan struct{} // 用于停止后台清理协程
}

// 定义minHeap类型，实现heap.Interface接口
type minHeap []*entry

func (h minHeap) Len() int { return len(h) }

func (h minHeap) Less(i, j int) bool {
	return h[i].freq < h[j].freq
}

func (h minHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *minHeap) Push(x interface{}) {
	n := len(*h)
	ent := x.(*entry)
	ent.index = n
	*h = append(*h, ent)
}

func (h *minHeap) Pop() interface{} {
	old := *h
	n := len(old)
	ent := old[n-1]
	old[n-1] = nil // 避免内存泄漏
	ent.index = -1 // for safety
	*h = old[0 : n-1]
	return ent
}

// entry 存储键值对和访问信息
type entry struct {
	key       interface{} // 缓存键
	value     interface{} // 缓存值
	freq      int         // 访问频率计数器
	index     int         // 在堆中的索引位置
	expiresAt time.Time   // 过期时间
}

// NewLFUCache 创建LFU缓存实例
// capacity: 缓存最大容量
func NewLFUCache(capacity int) *LFUCache {
	c := &LFUCache{
		capacity: capacity,
		cache:    make(map[interface{}]*entry, capacity+1), // 预分配空间减少扩容
		heap:     &minHeap{},
		stopChan: make(chan struct{}),
	}
	// 启动后台清理协程，定期清理过期条目
	go c.startCleaner(1 * time.Minute)
	return c
}

// Get 获取缓存值
// 1. 检查键是否存在
// 2. 检查是否过期
// 3. 增加访问频率并调整堆
// 4. 更新统计信息
func (l *LFUCache) Get(key interface{}) (interface{}, bool) {
	l.lock.RLock()
	ent, ok := l.cache[key]
	l.lock.RUnlock()

	if !ok {
		l.lock.Lock()
		l.stats.misses++
		l.lock.Unlock()
		return nil, false
	}

	l.lock.Lock()
	defer l.lock.Unlock()

	if !ent.expiresAt.IsZero() && time.Now().After(ent.expiresAt) {
		delete(l.cache, key)
		heap.Remove(l.heap, ent.index)
		l.stats.misses++
		l.stats.expiredCount++
		return nil, false
	}

	ent.freq++
	heap.Fix(l.heap, ent.index)
	l.stats.hits++
	return ent.value, true
}

// PutWithExpiration 添加/更新缓存(带过期时间)
// 1. 已存在则更新值和过期时间
// 2. 不存在则添加新条目
// 3. 容量满时淘汰频率最低的条目
func (l *LFUCache) PutWithExpiration(key, value interface{}, expiration time.Duration) {
	l.lock.Lock()
	defer l.lock.Unlock()

	var expiresAt time.Time
	if expiration > 0 {
		expiresAt = time.Now().Add(expiration)
	}

	if ent, ok := l.cache[key]; ok {
		ent.value = value
		ent.freq++
		ent.expiresAt = expiresAt
		heap.Fix(l.heap, ent.index)
		return
	}

	if len(l.cache) >= l.capacity {
		oldest := heap.Pop(l.heap).(*entry)
		delete(l.cache, oldest.key)
		l.stats.evictions++
	}

	ent := &entry{
		key:       key,
		value:     value,
		freq:      1,
		expiresAt: expiresAt,
	}
	heap.Push(l.heap, ent)
	l.cache[key] = ent
}

// startCleaner 启动后台清理协程
// interval: 清理间隔时间
func (l *LFUCache) startCleaner(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			l.Cleanup()
		case <-l.stopChan:
			return
		}
	}
}

func (l *LFUCache) Close() {
	close(l.stopChan)
}

// Cleanup 清理过期缓存条目
// 返回清理的条目数量
func (l *LFUCache) Cleanup() int {
	l.lock.Lock()
	defer l.lock.Unlock()

	count := 0
	for i := 0; i < l.heap.Len(); i++ {
		ent := (*l.heap)[i]
		if !ent.expiresAt.IsZero() && time.Now().After(ent.expiresAt) {
			delete(l.cache, ent.key)
			heap.Remove(l.heap, i)
			count++
			i--
		}
	}
	l.stats.expiredCount += int64(count)
	return count
}

func (l *LFUCache) Stats() (hits, misses, evictions, expired int64) {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return l.stats.hits, l.stats.misses, l.stats.evictions, l.stats.expiredCount
}

func (l *LFUCache) HitRate() float64 {
	hits, misses, _, _ := l.Stats()
	total := hits + misses
	if total == 0 {
		return 0
	}
	return float64(hits) / float64(total)
}

func (l *LFUCache) Put(key, value interface{}) {
	l.lock.Lock()
	defer l.lock.Unlock()

	// 如果键已存在，更新值并增加频率
	if ent, ok := l.cache[key]; ok {
		ent.value = value
		ent.freq++
		heap.Fix(l.heap, ent.index)
		return
	}

	// 如果缓存已满，淘汰使用频率最低的项
	if len(l.cache) >= l.capacity {
		oldest := heap.Pop(l.heap).(*entry)
		delete(l.cache, oldest.key)
	}

	// 添加新项
	ent := &entry{key: key, value: value, freq: 1}
	heap.Push(l.heap, ent)
	l.cache[key] = ent
}

// Len 获取当前缓存大小
func (l *LFUCache) Len() int {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return len(l.cache)
}

// Clear 清空缓存
func (l *LFUCache) Clear() {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.cache = make(map[interface{}]*entry)
	l.heap = &minHeap{}
}

func main() {
	cache := NewLFUCache(3)
	defer cache.Close()

	// 测试用例展示LFU特性
	cache.PutWithExpiration("A", 1, 0)
	cache.PutWithExpiration("B", 2, 0)
	cache.PutWithExpiration("C", 3, 0)

	cache.Get("A") // A频率=2
	cache.Get("A") // A频率=3
	cache.Get("B") // B频率=2

	cache.PutWithExpiration("D", 4, 0) // 应该淘汰C(频率最低)

	fmt.Printf("命中率: %.2f%%\n", cache.HitRate()*100)
}

func TestLFU(t *testing.T) {
	cache := NewLFUCache(2)

	// 测试1: 基本功能
	cache.Put("X", 10)
	if val, _ := cache.Get("X"); val != 10 {
		t.Error("基本功能测试失败")
	}

	// 测试2: LFU淘汰策略
	cache.Put("Y", 20)
	cache.Get("X")     // X频率=2
	cache.Put("Z", 30) // 应该淘汰Y(频率=1)

	if _, ok := cache.Get("Y"); ok {
		t.Error("LFU淘汰策略失败")
	}

	// 测试3: 过期功能
	cache.PutWithExpiration("T", "temp", time.Millisecond*50)
	time.Sleep(time.Millisecond * 100)
	if _, ok := cache.Get("T"); ok {
		t.Error("过期检查失败")
	}
}
