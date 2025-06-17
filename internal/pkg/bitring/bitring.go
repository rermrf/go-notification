package bitring

import "sync"

const (
	// bitsPerWord 表示一个uint64的位数
	bitsPerWord = 64
	// bitMask 用于位操作的掩码(63 = 0x3f)
	bitsMask = bitsPerWord - 1
	// bitsShift 用于计算位的偏移量(6 = log2(64))
	bitsShift = 6

	// 默认值
	defaultSize        = 128
	defaultConsecutive = 3
)

// BitRing 以位方式记录事件的滑动窗口
type BitRing struct {
	words       []int64      // 实际存储
	size        int          // 窗口长度
	pos         int          // 下一写入位置
	filled      bool         // 标记环形缓冲区是否已满（完成一轮）
	eventCount  int          // 当前窗口内事件发生数
	threshold   float64      // 事件发生率阈值
	consecutive int          // 连续事物触发次数
	mu          sync.RWMutex // 保证并发安全
}

// NewBitRing 创建一个新的BitRing
// size: 滑动窗口大小
// threshold: 事件
func NewBitRing(size int, threshold float64, consecutive int) *BitRing {
	if size <= 0 {
		size = defaultSize
	}
	if consecutive <= 0 {
		consecutive = defaultConsecutive
	}
	if consecutive > size {
		consecutive = size
	}
	// 确保阈值在有效范围
	if threshold < 0 {
		threshold = 0
	} else if threshold > 1 {
		threshold = 1
	}
	return &BitRing{
		words:       make([]int64, (size+bitsMask)/bitsPerWord),
		size:        size,
		threshold:   threshold,
		consecutive: consecutive,
	}
}

// Add 记录一次结果；eventHappened=true 表示事件发生
func (br *BitRing) Add(eventHappened bool) {
	br.mu.Lock()
	defer br.mu.Unlock()

	oldBit := br.bitAt(br.pos)

	// 更新计数
	if br.filled && oldBit {
		br.eventCount--
	}
	br.setBit(br.pos, eventHappened)
	if eventHappened {
		br.eventCount++
	}

	// 前进指针
	br.pos++
	if br.pos == br.size {
		br.pos = 0
		br.filled = true
	}
}

// bitAt 获取指定位置的bit值
func (br *BitRing) bitAt(idx int) bool {
	word := idx >> bitsShift   // 等价于 idx / 64
	off := int(idx & bitsMask) // 等价于idx % 64
	return (br.words[word]>>off)&1 == 1
}

// setBit 设置指定位置的bit值
func (br *BitRing) setBit(idx int, val bool) {
	word := idx >> bitsShift
	off := int(idx & bitsMask)
	if val {
		br.words[word] |= 1 << off
	} else {
		br.words[word] &^= 1 << off
	}
}
