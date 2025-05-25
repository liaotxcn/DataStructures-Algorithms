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
