package主干

import (
	"container/list"
	"fmt"
	"sync"
)

// ARC Cache 自适应替换缓存结构
// 使用4个链表实现ARC算法：
// t1: 存储最近访问的条目(新加入的条目)
// t2: 存储频繁访问的条目(从t1晋升过来的条目)
// b1: 存储从t1淘汰的幽灵条目(记录淘汰历史)
// b2: 存储从t2淘汰的幽灵条目(记录淘汰历史)
type ARCCache struct {
	capacity int // 缓存总容量
	p        int // 自适应参数，决定t1和t2的平衡点

	t1 *list.List // 最近访问的条目链表(LRU顺序)
	b1 *list.List // 从t1淘汰的幽灵条目链表
	t2 *list.List // 频繁访问的条目链表(LRU顺序)
	b2 *list.List // 从t2淘汰的幽灵条目链表

	lookup map[interface{}]*list.Element // 哈希表，用于快速查找
	lock   sync.RWMutex                  // 读写锁，保证线程安全

	stats struct { // 运行时统计信息
		hits      int64 // 命中次数
		misses    int64 // 未命中次数
		evictions int64 // 淘汰次数
	}
}

// entry 缓存条目结构
type entry struct {
	key   interface{} // 缓存键
	value interface{} // 缓存值
	ghost bool        // 是否为幽灵条目(仅记录淘汰历史)
}

// NewARCCache 创建新的ARC缓存实例
// capacity: 缓存容量，决定t1+t2的最大长度
func NewARCCache(capacity int) *ARCCache {
	return &ARCCache{
		capacity: capacity,
		t1:       list.New(), // 初始化空链表
		b1:       list.New(),
		t2:       list.New(),
		b2:       list.New(),
		lookup:   make(map[interface{}]*list.Element), // 预分配哈希表
	}
}

// Get 获取缓存值
// 1. 检查键是否存在
// 2. 如果是幽灵条目则返回未命中
// 3. 如果在t1中则晋升到t2
// 4. 返回值和命中状态
func (a *ARCCache) Get(key interface{}) (interface{}, bool) {
	a.lock.Lock()
	defer a.lock.Unlock()

	if elem, ok := a.lookup[key]; ok {
		ent := elem.Value.(*entry)
		if ent.ghost {
			a.stats.misses++
			return nil, false
		}

		// 如果在t1中，移动到t2(晋升为频繁访问)
		if a.isInList(elem, a.t1) {
			a.t1.Remove(elem)
			elem = a.t2.PushFront(ent) // 移动到t2头部
			a.lookup[key] = elem
		}
		a.stats.hits++
		return ent.value, true
	}
	a.stats.misses++
	return nil, false
}

// Put 添加或更新缓存
// 1. 如果键已存在则更新值
// 2. 如果在幽灵队列中则调整p值
// 3. 添加新条目到t1
// 4. 如果超过容量则执行替换
func (a *ARCCache) Put(key, value interface{}) {
	a.lock.Lock()
	defer a.lock.Unlock()

	// 如果key已存在，更新值
	if elem, ok := a.lookup[key]; ok {
		ent := elem.Value.(*entry)
		if !ent.ghost {
			ent.value = value
			// 如果在t1中，移动到t2
			if a.isInList(elem, a.t1) {
				a.t1.Remove(elem)
				elem = a.t2.PushFront(ent)
				a.lookup[key] = elem
			}
			return
		}
	}

	// 如果key在b1中，增加p
	if elem, ok := a.lookup[key]; ok && a.isInList(elem, a.b1) {
		a.p = min(a.capacity, a.p+max(1, a.b2.Len()/a.b1.Len()))
		a.replace(false)
		a.b1.Remove(elem)
		delete(a.lookup, key)
	}

	// 如果key在b2中，减少p
	if elem, ok := a.lookup[key]; ok && a.isInList(elem, a.b2) {
		a.p = max(0, a.p-max(1, a.b1.Len()/a.b2.Len()))
		a.replace(true)
		a.b2.Remove(elem)
		delete(a.lookup, key)
	}

	// 添加新条目到t1
	ent := &entry{key: key, value: value}
	elem := a.t1.PushFront(ent)
	a.lookup[key] = elem

	// 如果总长度超过容量，执行替换
	if a.t1.Len()+a.t2.Len() >= a.capacity {
		a.replace(false)
	}
}

// replace 执行替换策略
// 根据p值决定从t1还是t2淘汰条目
// inB2: 是否因为访问b2中的幽灵条目而触发替换
func (a *ARCCache) replace(inB2 bool) {
	// 如果t1不为空且(t1长度大于p 或 因访问b2且t1长度等于p)
	if a.t1.Len() > 0 && (a.t1.Len() > a.p || (inB2 && a.t1.Len() == a.p)) {
		// 从t1淘汰最久未访问的条目
		if elem := a.t1.Back(); elem != nil {
			a.stats.evictions++
			ent := a.t1.Remove(elem).(*entry)
			ent.ghost = true           // 转为幽灵条目
			elem = a.b1.PushFront(ent) // 加入b1记录淘汰历史
			a.lookup[ent.key] = elem
		}
	} else {
		// 否则从t2淘汰最久未访问的条目
		if elem := a.t2.Back(); elem != nil {
			a.stats.evictions++
			ent := a.t2.Remove(elem).(*entry)
			ent.ghost = true
			elem = a.b2.PushFront(ent) // 加入b2记录淘汰历史
			a.lookup[ent.key] = elem
		}
	}
}

// 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// 优化isInList方法
func (a *ARCCache) isInList(elem *list.Element, l *list.List) bool {
	// 通过直接比较指针地址快速判断
	for e := l.Front(); e != nil; e = e.Next() {
		if e == elem {
			return true
		}
	}
	return false
}

// 修改main函数测试用例
func main() {
	cache := NewARCCache(3)

	// 添加更多测试操作
	cache.Put("A", 1)
	cache.Put("B", 2)
	cache.Put("C", 3)

	// 多次访问测试命中率
	cache.Get("A")
	cache.Get("B")
	cache.Get("A")

	// 测试未命中
	cache.Get("X")

	// 触发淘汰
	cache.Put("D", 4)
	cache.Put("E", 5)
	printCache(cache)

	fmt.Println("\n=== 阶段4: 幽灵条目影响 ===")
	cache.Put("C", 3) // 重新插入被淘汰的C
	printCache(cache)

	fmt.Println("\n=== 统计信息 ===")
	fmt.Printf("命中次数: %d\n", cache.stats.hits)
	fmt.Printf("未命中次数: %d\n", cache.stats.misses)
	fmt.Printf("淘汰次数: %d\n", cache.stats.evictions)
}

// printCache 打印当前缓存内容
func printCache(c *ARCCache) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	fmt.Print("T1(最近访问): ")
	for e := c.t1.Front(); e != nil; e = e.Next() {
		ent := e.Value.(*entry)
		fmt.Printf("%v(%v) ", ent.key, ent.value)
	}

	fmt.Print("\nT2(频繁访问): ")
	for e := c.t2.Front(); e != nil; e = e.Next() {
		ent := e.Value.(*entry)
		fmt.Printf("%v(%v) ", ent.key, ent.value)
	}

	fmt.Print("\nB1(最近淘汰): ")
	for e := c.b1.Front(); e != nil; e = e.Next() {
		ent := e.Value.(*entry)
		fmt.Printf("%v ", ent.key)
	}

	fmt.Print("\nB2(频繁淘汰): ")
	for e := c.b2.Front(); e != nil; e = e.Next() {
		ent := e.Value.(*entry)
		fmt.Printf("%v ", ent.key)
	}
	fmt.Println()
}
