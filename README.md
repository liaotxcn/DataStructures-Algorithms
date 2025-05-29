# DataStructures-Algorithms 数构与算法 📊  

<div align="center">  

![Python](https://img.shields.io/badge/Python-3776AB?style=for-the-badge&logo=python&logoColor=white)
![Go](https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white)  

</div>  

---

## 📂 项目结构  

### **Python 版本** 🐍  
- **`First.py`**  
  - 基础数据结构实现  
  - 常用算法模板和实例  
- **`Second.py`**  
  - 高级数据结构  
  - 进阶算法实现、应用案例

### **Golang 版本** 🦫  
- **`First.go`**  
  - 基础数据结构(链表、队列、双端队列、栈、集合、图等)
  - 常用算法(排序、搜索、动态规划等)
  - 工具实例(字符串处理、切片操作等)
- **`Second.go`**  
  - 并发安全数据结构(Map、Slice、Queue等)
  - Goroutine管理(工作池、优雅退出等模式)
  - Channel高级模式(Fan-in、Fan-out、超时控制等)
  - 原子操作与并发原语(计数器、Once等)
  - 经典并发模式(生产者消费者等)
- **`Third.go`**  
  - 高级Goroutine模式、Channel、并发实用
  - 分片并发安全Map实现(Set、Get、getShard)
  - 并发安全环形缓冲区(Put、Get)
  - 动态Worker Pool实现
  - 高级并发模式(Context、Channel多路复用、限速器、ErrorGroup增强版)
  - 工具函数实例(FNV32哈希算法实现、并发原语等)
- **`Cache elimination 缓存淘汰`**
  - FIFO   按照数据最早进入顺序淘汰数据
  - LRU    根据数据最近使用情况淘汰数据
  - LFU    根据数据访问频率来淘汰数据
  - ARC    LRU + LFU

---

## 缓存淘汰(FIFO、LRU、LFU、ARC)
### FIFO(先进先出)
- **原理**：优先淘汰最早进入缓存的数据
- **实现方式**：使用队列记录数据进入顺序
- **特点**：
  - ✅ 实现简单，内存开销低  
  - ❌ 可能误删高频访问的早期数据
- **示例**（容量=3）：  
  访问序列：A → B → C → A → D  
  淘汰顺序：B（最早进入且未重复访问）
![image](https://github.com/user-attachments/assets/fe7b6a29-a622-4471-8267-7d2afb419c8f)

### LRU(最近最少使用)
- **原理**：淘汰最久未被访问的数据
- **实现方式**：哈希表+双向链表（O(1)复杂度）
- **特点**：
  - ✅ 符合时间局部性原理  
  - ❌ 突发流量可能挤出热点数据
- **示例**（容量=3）：  
  访问序列：A → B → C → A → D → B → E  
  淘汰顺序：C（最久未访问）→ A → D
![image](https://github.com/user-attachments/assets/6a1fc97e-e35e-4b10-bad7-646642013ca9)

### LFU(最不经常使用)
- **原理**：淘汰访问频率最低的数据（频率相同则按LRU）
- **实现方式**：最小堆/多层链表+频率哈希表
- **特点**：
  - ✅ 适合长期热点场景  
  - ❌ 容易积累"缓存污染"数据
- **示例**（容量=2）：  
  访问序列：A → A → B → A → C → B → C  
  淘汰顺序：B（频率1）→ A（频率3保留）
![image](https://github.com/user-attachments/assets/fa84f11a-35bb-4379-ada2-f6c06922946b)

### ARC(自适应替换缓存)
- **原理**：动态平衡LRU和LFU策略
- **实现方式**：维护4个队列（T1/T2/B1/B2）
- **特点**：
  - ✅ 自适应各种访问模式  
  - ❌ 实现复杂，内存占用高
- **工作模式**：根据命中率自动调整LRU/LFU权重
![image](https://github.com/user-attachments/assets/ebe32488-0c6b-4886-8770-fa54ea3ac42e)

### 对比
| 算法     | 时间复杂度 | 空间复杂度 | 最佳适用场景         |
|----------|------------|------------|----------------------|
| FIFO     | O(1)       | O(n)       | 顺序访问场景         |
| LRU      | O(1)       | O(n)       | 短期热点数据         |
| LFU      | O(log n)   | O(n)       | 长期稳定热点         |
| ARC      | O(1)       | O(n)       | 复杂多变访问模式     |

### 选型
- **低成本实现**：FIFO
- **通用场景**：LRU（Redis默认策略(改进LRU)）
- **稳定热点**：LFU
- **高性能系统**：ARC（数据库缓存常用）
## Redis改进版LRU、LFU
### LRU
- 传统LRU算法基于链表实现，链表中元素按照顺序从前到后排列，最新操作的键会被移动到表头，进行内存淘汰时，则删除链表尾部元素(即最久未被使用的元素)
- 传统LRU弊端1：需要用链表管理所有缓存数据，带来额外空间开销
- 传统LRU弊端2：当有数据被访问时，需在链表上将该数据移动到表头位置，若有大量数据被访问，就会带来大量的链表移动操作，时间耗费长，进而降低Redis缓存性能
- Redis改进LRU算法：在Redis对象结构体中添加一个额外的字段，用于记录此数据最后一次访问时间，进行内存淘汰时，采用随机采样（默认取5个键）而非全局排序，从中淘汰最久未使用的键，以减少性能开销
- 配置参数：maxmemory-policy allkeys-lru（所有键参与淘汰）
           maxmemory-policy volatile-lru（仅带过期时间的键参与淘汰）
- 优点：不用为所有数据维护一个大表链，节省空间占用；不用在每次数据访问时都移动链表项，提升缓存性能，适合大多数访问模式(如热点数据场景)
- 存在无法解决缓存污染的问题(一次读取大量数据，且只会被读取一次，大量数据长时间留存Redis)
### LFU
- 频率统计：基于计数器（Morris 计数器）近似统计访问频率，节省内存
- 衰减机制：计数器随时间衰减（通过 lfu-decay-time 配置），避免旧数据长期占据内存
- 配置参数：maxmemory-policy allkeys-lfu 或 volatile-lfu
           lfu-log-factor 调整计数器增长速率（值越大，区分低频访问越精细）
- 优点：更适合长尾访问分布（如某些数据偶尔访问但不应保留）
- 对突发访问敏感，新数据可能因初始频率低被误删
### 场景选择
- LRU：适合有明显热点数据的场景（如用户近期访问记录）
- LFU：适合访问模式均匀或需要长期频率统计的场景（如缓存广告数据）
- 其他策略：若无需精确淘汰，可结合 TTL（如 volatile-ttl）或随机淘汰（allkeys-random）
```bash
# 使用 LFU 淘汰所有键
maxmemory-policy allkeys-lfu
lfu-log-factor 10
lfu-decay-time 1

# 使用 LRU 淘汰仅带过期时间的键
maxmemory-policy volatile-lru
```
- 采样数量：通过 maxmemory-samples 调整 LRU/LFU 的采样精度（增加样本数提高准确性，但消耗更多CPU）
- 混合策划：Redis 6.2+ 支持 volatile-lfu 和 volatile-lru 对带过期时间的键灵活控制

---

## 🚀 快速开始  
```bash
git clone https://github.com/liaotxcn/DataStructures-Algorithms.git  # 克隆仓库
```
```bash
# 进入目录
cd DataStructures-Algorithms/
cd Python/ 
cd Golang/
```
```bash
python First.py  
go run First.go
```

### 持续更新中...
