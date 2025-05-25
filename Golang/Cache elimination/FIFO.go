package主干

import (
	"container/list"
	"fmt"
	"sync"
	"time"
)

// FIFO Cache 线程安全的FIFO缓存结构
// 使用哈希表+双向链表实现，哈希表提供O(1)访问，链表维护FIFO顺序
type FIFOCache struct {
	capacity int                    // 缓存最大容量
	cache    map[interface{}]*entry // 哈希表存储键和entry指针
	queue    *list.List             // 双向链表，维护插入顺序(FIFO)
	lock     sync.RWMutex           // 读写锁，保证线程安全
	stats    struct {               // 运行时统计信息
		hits   int64 // 命中次数
		misses int64 // 未命中次数
	}
	stopChan chan struct{} // 用于停止后台清理协程
}

// entry 缓存条目结构
type entry struct {
	key       interface{} // 缓存键
	value     interface{} // 缓存值
	expiresAt time.Time   // 过期时间(零值表示永不过期)
}

// NewFIFOCache 创建新的FIFO缓存实例
// capacity: 缓存容量
func NewFIFOCache(capacity int) *FIFOCache {
	c := &FIFOCache{
		capacity: capacity,
		cache:    make(map[interface{}]*entry, capacity+1), // 预分配空间减少扩容
		queue:    list.New(),                               // 初始化双向链表
		stopChan: make(chan struct{}),                      // 初始化停止通道
	}
	// 启动后台协程定期清理过期条目
	go c.startCleaner(1 * time.Minute)
	return c
}

// Get 获取缓存值
// 1. 检查键是否存在
// 2. 检查是否过期(过期则删除)
// 3. 返回值和状态
func (f *FIFOCache) Get(key interface{}) (interface{}, bool) {
	f.lock.RLock()
	elem, ok := f.cache[key]
	f.lock.RUnlock()

	if !ok {
		f.lock.Lock()
		f.stats.misses++
		f.lock.Unlock()
		return nil, false
	}

	f.lock.Lock()
	defer f.lock.Unlock()

	if !elem.expiresAt.IsZero() && time.Now().After(elem.expiresAt) {
		delete(f.cache, key)
		f.stats.misses++
		return nil, false
	}

	f.stats.hits++
	return elem.value, true
}

// PutWithExpiration 添加带过期时间的缓存
// 1. 已存在则更新值和过期时间
// 2. 不存在则添加新条目
// 3. 缓存满时淘汰最早进入的项(FIFO)
func (f *FIFOCache) PutWithExpiration(key, value interface{}, expiration time.Duration) {
	f.lock.Lock()
	defer f.lock.Unlock()

	var expiresAt time.Time
	if expiration > 0 {
		expiresAt = time.Now().Add(expiration)
	}

	// 如果键已存在，更新值
	if elem, ok := f.cache[key]; ok {
		elem.value = value
		elem.expiresAt = expiresAt
		return
	}

	// 如果缓存已满，淘汰最早进入的项
	if len(f.cache) >= f.capacity {
		oldest := f.queue.Front()
		if oldest != nil {
			delete(f.cache, oldest.Value.(*entry).key)
			f.queue.Remove(oldest)
		}
	}

	// 添加新项到链表尾部
	elem := &entry{
		key:       key,
		value:     value,
		expiresAt: expiresAt,
	}
	f.queue.PushBack(elem)
	f.cache[key] = elem
}

// Put 添加缓存
func (f *FIFOCache) Put(key, value interface{}) {
	f.PutWithExpiration(key, value, 0)
}

// Len 获取当前缓存大小
func (f *FIFOCache) Len() int {
	f.lock.RLock()
	defer f.lock.RUnlock()
	return len(f.cache)
}

// Clear 清空缓存
func (f *FIFOCache) Clear() {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.cache = make(map[interface{}]*entry)
	f.queue = list.New()
}

// startCleaner 启动后台清理过期条目的协程
// interval: 清理间隔时间
func (f *FIFOCache) startCleaner(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			f.cleanupExpired()
		case <-f.stopChan:
			return
		}
	}
}

// cleanupExpired 清理过期条目
// 遍历链表，删除所有已过期的条目
// 返回清理的条目数量
func (f *FIFOCache) cleanupExpired() int {
	f.lock.Lock()
	defer f.lock.Unlock()

	count := 0
	var next *list.Element
	for e := f.queue.Front(); e != nil; e = next {
		next = e.Next() // 先获取下一个元素
		ent := e.Value.(*entry)
		if !ent.expiresAt.IsZero() && time.Now().After(ent.expiresAt) {
			delete(f.cache, ent.key)
			f.queue.Remove(e)
			count++
		}
	}
	return count
}

// Close 关闭缓存，停止后台清理协程
func (f *FIFOCache) Close() {
	close(f.stopChan)
}

func main() {
	cache := NewFIFOCache(3)
	defer cache.Close()

	// 初始填充缓存
	fmt.Println("=== 初始填充缓存 ===")
	cache.Put("A", 1)
	cache.Put("B", 2)
	cache.Put("C", 3)
	printCache(cache) // 输出: A(1) B(2) C(3)

	// 测试FIFO淘汰策略
	fmt.Println("\n=== 测试FIFO淘汰 ===")
	cache.Put("D", 4) // 应该淘汰最早进入的A
	printCache(cache)

	// 测试过期功能
	fmt.Println("\n=== 测试过期功能 ===")
	cache.PutWithExpiration("E", 5, 2*time.Second)
	printCache(cache)
	time.Sleep(3 * time.Second)
	if _, ok := cache.Get("E"); !ok {
		fmt.Println("E已过期")
	}
	printCache(cache)
}

// printCache 打印当前缓存内容
func printCache(c *FIFOCache) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	fmt.Print("当前缓存: ")
	for e := c.queue.Front(); e != nil; e = e.Next() {
		ent := e.Value.(*entry)
		if ent.expiresAt.IsZero() || time.Now().Before(ent.expiresAt) {
			fmt.Printf("%v(%v) ", ent.key, ent.value)
		}
	}
	fmt.Println()
}

