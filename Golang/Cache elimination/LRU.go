package main

import (
	"container/list"
	"fmt"
	"sync"
	"time"
)

// LRUCache 线程安全的LRU缓存结构
// 使用哈希表+双向链表实现，哈希表提供O(1)访问，链表维护访问顺序
type LRUCache struct {
	capacity   int                           // 缓存最大容量
	cache      map[interface{}]*list.Element // 哈希表存储键和链表节点指针
	list       *list.List                    // 双向链表，头部最新尾部最旧
	lock       sync.RWMutex                  // 读写锁，支持并发读写
	expiration time.Duration                 // 全局默认过期时间
	stats      struct {                      // 运行时统计信息
		hits         int64 // 缓存命中次数
		misses       int64 // 缓存未命中次数
		evictions    int64 // 因容量淘汰的条目数
		expiredCount int64 // 因过期淘汰的条目数
	}
}

// entry 链表节点数据结构
// 存储键值对和过期时间信息
type entry struct {
	key       interface{} // 缓存键
	value     interface{} // 缓存值
	expiresAt time.Time   // 过期时间戳，零值表示永不过期
}

// NewLRUCache 构造函数
// capacity: 缓存最大容量
// expiration: 全局默认过期时间，0表示永不过期
func NewLRUCache(capacity int, expiration time.Duration) *LRUCache {
	return &LRUCache{
		capacity:   capacity,
		cache:      make(map[interface{}]*list.Element, capacity), // 预分配空间
		list:       list.New(),                                    // 初始化双向链表
		expiration: expiration,
	}
}

// Get 获取缓存值
// 1. 检查键是否存在
// 2. 检查是否过期
// 3. 更新访问时间(移动到链表头部)
// 4. 更新统计信息
func (l *LRUCache) Get(key interface{}) (interface{}, bool) {
	l.lock.RLock()
	elem, ok := l.cache[key]
	if !ok {
		l.lock.RUnlock()
		l.lock.Lock()
		l.stats.misses++
		l.lock.Unlock()
		return nil, false
	}

	ent := elem.Value.(*entry)
	if !ent.expiresAt.IsZero() && time.Now().After(ent.expiresAt) {
		l.lock.RUnlock()
		l.lock.Lock()
		delete(l.cache, key)
		l.list.Remove(elem)
		l.stats.misses++
		l.stats.expiredCount++
		l.lock.Unlock()
		return nil, false
	}
	l.lock.RUnlock()

	l.lock.Lock()
	l.list.MoveToFront(elem)
	l.stats.hits++
	l.lock.Unlock()
	return ent.value, true
}

// Put 添加/更新缓存(使用默认过期时间)
func (l *LRUCache) Put(key, value interface{}) {
	l.PutWithExpiration(key, value, l.expiration)
}

// PutWithExpiration 添加/更新缓存(自定义过期时间)
// 1. 已存在则更新值和过期时间
// 2. 不存在则添加新条目
// 3. 容量满时淘汰最久未使用的条目
func (l *LRUCache) PutWithExpiration(key, value interface{}, expiration time.Duration) {
	l.lock.Lock()
	defer l.lock.Unlock()

	var expiresAt time.Time
	if expiration > 0 {
		expiresAt = time.Now().Add(expiration)
	}

	// 如果键已存在，更新值并移动到链表头部
	if elem, ok := l.cache[key]; ok {
		ent := elem.Value.(*entry)
		ent.value = value
		ent.expiresAt = expiresAt
		l.list.MoveToFront(elem)
		return
	}

	// 如果缓存已满，淘汰最久未使用的项
	if len(l.cache) >= l.capacity {
		oldest := l.list.Back()
		if oldest != nil {
			delete(l.cache, oldest.Value.(*entry).key)
			l.list.Remove(oldest)
			l.stats.evictions++
		}
	}

	// 添加新项到链表头部并存入哈希表
	elem := l.list.PushFront(&entry{
		key:       key,
		value:     value,
		expiresAt: expiresAt,
	})
	l.cache[key] = elem
}

// Stats 获取缓存命中统计
func (l *LRUCache) Stats() (hits, misses, evictions, expired int64) {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return l.stats.hits, l.stats.misses, l.stats.evictions, l.stats.expiredCount
}

// Cleanup 主动清理过期缓存
// 从链表尾部开始检查(最久未使用)
// 返回清理的条目数量
func (l *LRUCache) Cleanup() int {
	l.lock.Lock()
	defer l.lock.Unlock()

	count := 0
	for elem := l.list.Back(); elem != nil; elem = l.list.Back() {
		ent := elem.Value.(*entry)
		if ent.expiresAt.IsZero() || time.Now().Before(ent.expiresAt) {
			break
		}
		delete(l.cache, ent.key)
		l.list.Remove(elem)
		count++
	}
	l.stats.expiredCount += int64(count)
	return count
}

// Len 获取当前缓存大小
func (l *LRUCache) Len() int {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return len(l.cache)
}

// Clear 清空缓存
func (l *LRUCache) Clear() {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.cache = make(map[interface{}]*list.Element)
	l.list = list.New()
	l.stats = struct {
		hits         int64
		misses       int64
		evictions    int64
		expiredCount int64
	}{}
}

func main() {
	// 创建容量为3，默认过期10秒的缓存
	cache := NewLRUCache(3, 10*time.Second)

	// 基本操作演示
	cache.Put("name", "PaiCloud")
	cache.Put("age", 25)
	cache.Put("job", "Engineer")

	// 获取存在的值
	if val, ok := cache.Get("name"); ok {
		fmt.Printf("name: %v\n", val)
	}

	// 触发淘汰（容量已满时添加新条目）
	cache.Put("salary", 50000) // 淘汰最久未使用的"age"

	// 检查被淘汰的键
	if _, ok := cache.Get("age"); !ok {
		fmt.Println("age已被淘汰") // 输出: age已被淘汰
	}

	// 过期功能演示
	cache.PutWithExpiration("temp", "data", 2*time.Second)
	time.Sleep(3 * time.Second)
	if _, ok := cache.Get("temp"); !ok {
		fmt.Println("temp已过期") // 输出: temp已过期
	}

	// 统计信息
	hits, misses, evictions, expired := cache.Stats()
	fmt.Printf("命中率: %.1f%%\n", float64(hits)/float64(hits+misses)*100)
	fmt.Printf("淘汰次数: %d, 过期次数: %d\n", evictions, expired)
}
