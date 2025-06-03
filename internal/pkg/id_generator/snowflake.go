package id_generator

const (
	// 位数分配常量
	timestampBits = 41 // 时间戳位数
	hashBits      = 10 // hash值位数
	sequenceBits  = 12 // 序列号位数

	// 位移常量
	sequenceShift  = 0
	hashShift      = sequenceBits
	timestampShift = sequenceBits + hashBits

	// 掩码常量
	sequenceMask  = (1 << sequenceBits) - 1
	hashMask      = (1 << hashBits) - 1
	timestampMask = (1 << timestampBits) - 1

	// 基准时间
)
